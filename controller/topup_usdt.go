package controller

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
)

type UsdtPayRequest struct {
	Amount float64 `json:"amount"`
}

// RequestUsdtPay creates a USDT top-up order via BEpusdt gateway.
func RequestUsdtPay(c *gin.Context) {
	if !setting.UsdtEnabled {
		c.JSON(200, gin.H{"message": "error", "data": "USDT 充值未启用"})
		return
	}

	var req UsdtPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(200, gin.H{"message": "error", "data": "参数错误"})
		return
	}

	if req.Amount < setting.UsdtMinTopUp {
		c.JSON(200, gin.H{"message": "error", "data": fmt.Sprintf("充值金额不能小于 %.2f", setting.UsdtMinTopUp)})
		return
	}

	id := c.GetInt("id")
	user, err := model.GetUserById(id, false)
	if err != nil || user == nil {
		c.JSON(200, gin.H{"message": "error", "data": "用户不存在"})
		return
	}

	orderID := service.GenerateUsdtOrderID()
	notifyURL := system_setting.ServerAddress + "/api/topup/usdt/callback"
	redirectURL := system_setting.ServerAddress

	resp, err := service.CreateBEpusdtOrder(orderID, req.Amount, notifyURL, redirectURL)
	if err != nil {
		log.Printf("USDT 创建订单失败: %v, UserId=%d", err, id)
		c.JSON(200, gin.H{"message": "error", "data": "充值订单创建失败"})
		return
	}

	expirationTime := resp.Data.ExpirationTime
	if expirationTime == 0 {
		expirationTime = time.Now().Unix() + int64(setting.UsdtOrderTimeout*60)
	}

	topUp := &model.TopUp{
		UserId:           id,
		Amount:           int64(req.Amount),
		Money:            req.Amount,
		TradeNo:          orderID,
		PaymentMethod:    "usdt",
		CreateTime:       common.GetTimestamp(),
		Status:           common.TopUpStatusPending,
		UsdtTradeId:      resp.Data.TradeId,
		UsdtActualAmount: resp.Data.ActualAmount,
		UsdtAddress:      resp.Data.Token,
		PaymentUrl:       resp.Data.PaymentUrl,
		ExpirationTime:   expirationTime,
	}
	if err := topUp.Insert(); err != nil {
		log.Printf("USDT 创建本地订单失败: %v, UserId=%d, OrderID=%s", err, id, orderID)
		c.JSON(200, gin.H{"message": "error", "data": "创建订单失败"})
		return
	}

	log.Printf("USDT 订单创建成功 - 用户: %d, 订单: %s, 金额: %.2f", id, orderID, req.Amount)

	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"payment_url":     resp.Data.PaymentUrl,
			"trade_no":        orderID,
			"usdt_amount":     resp.Data.ActualAmount,
			"expiration_time": expirationTime,
		},
	})
}

// UsdtCallback handles BEpusdt payment callback notifications.
func UsdtCallback(c *gin.Context) {
	contentType := c.GetHeader("Content-Type")
	if contentType == "" ||
		(!strings.Contains(contentType, "application/json") &&
			!strings.Contains(contentType, "application/x-www-form-urlencoded")) {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Parse callback parameters into a map
	params := make(map[string]string)
	if strings.Contains(contentType, "application/json") {
		var jsonBody map[string]interface{}
		if err := c.ShouldBindJSON(&jsonBody); err != nil {
			log.Printf("USDT 回调 JSON 解析失败: %v, IP=%s", err, c.ClientIP())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		for k, v := range jsonBody {
			params[k] = fmt.Sprintf("%v", v)
		}
	} else {
		if err := c.Request.ParseForm(); err != nil {
			log.Printf("USDT 回调 Form 解析失败: %v, IP=%s", err, c.ClientIP())
			c.AbortWithStatus(http.StatusBadRequest)
			return
		}
		for k, v := range c.Request.PostForm {
			if len(v) > 0 {
				params[k] = v[0]
			}
		}
	}

	signature := params["signature"]
	if signature == "" {
		log.Printf("USDT 回调签名为空, IP=%s", c.ClientIP())
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	// Verify signature
	if !service.VerifyBEpusdtSignature(params, signature, setting.UsdtApiAuthToken) {
		log.Printf("[安全告警] USDT 回调签名验证失败, IP=%s, params=%v", c.ClientIP(), params)
		c.AbortWithStatus(http.StatusForbidden)
		return
	}

	orderID := params["order_id"]
	statusStr := params["status"]
	tradeID := params["trade_id"]

	log.Printf("USDT 回调 - trade_id=%s, order_id=%s, status=%s, IP=%s", tradeID, orderID, statusStr, c.ClientIP())

	// Look up the order
	topUp := model.GetTopUpByTradeNo(orderID)
	if topUp == nil {
		log.Printf("[告警] USDT 回调 order_id 不存在: %s, trade_id=%s, IP=%s", orderID, tradeID, c.ClientIP())
		c.String(http.StatusOK, "ok")
		return
	}

	// Idempotent: already succeeded
	if topUp.Status == common.TopUpStatusSuccess {
		c.String(http.StatusOK, "ok")
		return
	}

	// Process payment success (status == 2)
	status, _ := strconv.Atoi(statusStr)
	if status == 2 {
		LockOrder(orderID)
		defer UnlockOrder(orderID)

		if err := model.RechargeUsdt(orderID); err != nil {
			log.Printf("USDT 充值处理失败: %v, order_id=%s, trade_id=%s", err, orderID, tradeID)
		} else {
			log.Printf("USDT 充值成功 - order_id=%s, trade_id=%s", orderID, tradeID)
		}
	}

	c.String(http.StatusOK, "ok")
}

// GetUsdtOrderStatus queries the status of a USDT top-up order.
func GetUsdtOrderStatus(c *gin.Context) {
	tradeNo := c.Query("trade_no")
	if tradeNo == "" {
		c.JSON(200, gin.H{"message": "error", "data": "缺少 trade_no 参数"})
		return
	}

	topUp := model.GetTopUpByTradeNo(tradeNo)
	if topUp == nil {
		c.JSON(200, gin.H{"message": "error", "data": "订单不存在"})
		return
	}

	c.JSON(200, gin.H{
		"message": "success",
		"data": gin.H{
			"status":   topUp.Status,
			"trade_no": topUp.TradeNo,
		},
	})
}
