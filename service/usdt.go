package service

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
)

// BEpusdtOrderResponse represents the response from BEpusdt create-transaction API.
type BEpusdtOrderResponse struct {
	StatusCode int                    `json:"status_code"`
	Message    string                 `json:"message"`
	Data       BEpusdtOrderData       `json:"data"`
}

type BEpusdtOrderData struct {
	TradeId        string `json:"trade_id"`
	OrderId        string `json:"order_id"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	ActualAmount   string `json:"actual_amount"`
	ReceiveAddress string `json:"receive_address"`
	Token          string `json:"token"`
	ExpirationTime int64  `json:"expiration_time"`
	PaymentUrl     string `json:"payment_url"`
}

// BEpusdtCheckStatusResponse represents the response from BEpusdt check-status API.
type BEpusdtCheckStatusResponse struct {
	StatusCode int                        `json:"status_code"`
	Message    string                     `json:"message"`
	Data       BEpusdtCheckStatusData     `json:"data"`
}

type BEpusdtCheckStatusData struct {
	TradeId string `json:"trade_id"`
	Status  int    `json:"status"`
}

// ComputeBEpusdtSignature computes the BEpusdt signature for the given parameters.
// It collects non-empty params (excluding "signature"), sorts by key ASCII ascending,
// joins as key=value&key=value, appends authToken at the end, and computes lowercase MD5.
func ComputeBEpusdtSignature(params map[string]string, authToken string) string {
	// Collect keys of non-empty, non-signature params
	keys := make([]string, 0, len(params))
	for k, v := range params {
		if k == "signature" || v == "" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build the string to sign: key=value&key=value&...authToken
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('&')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(params[k])
	}
	sb.WriteString(authToken)

	hash := md5.Sum([]byte(sb.String()))
	return hex.EncodeToString(hash[:])
}

// VerifyBEpusdtSignature verifies a BEpusdt callback signature.
func VerifyBEpusdtSignature(params map[string]string, signature string, authToken string) bool {
	computed := ComputeBEpusdtSignature(params, authToken)
	return computed == signature
}

// CreateBEpusdtOrder calls the BEpusdt API to create a payment order.
// Compatible with v03413/BEpusdt which uses /api/v1/order/create-transaction
// and trade_type field (e.g. "usdt.trc20") instead of separate currency/token/network.
func CreateBEpusdtOrder(orderID string, amount float64, notifyURL string, redirectURL string) (*BEpusdtOrderResponse, error) {
	// Build trade_type from token + network, e.g. "usdt.trc20"
	tradeType := setting.UsdtToken + "." + setting.UsdtNetwork
	if setting.UsdtNetwork == "tron" {
		tradeType = setting.UsdtToken + ".trc20"
	}

	// String params for signature computation
	signParams := map[string]string{
		"order_id":     orderID,
		"amount":       fmt.Sprintf("%g", amount),
		"notify_url":   notifyURL,
		"redirect_url": redirectURL,
		"trade_type":   tradeType,
	}

	signature := ComputeBEpusdtSignature(signParams, setting.UsdtApiAuthToken)

	// Build JSON body with proper types (amount as float, not string)
	jsonBody := map[string]interface{}{
		"order_id":     orderID,
		"amount":       amount,
		"notify_url":   notifyURL,
		"redirect_url": redirectURL,
		"trade_type":   tradeType,
		"signature":    signature,
	}

	body, err := common.Marshal(jsonBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %v", err)
	}

	apiURL := strings.TrimRight(setting.UsdtApiUrl, "/") + "/api/v1/order/create-transaction"

	common.SysLog(fmt.Sprintf("BEpusdt create order: url=%s, order_id=%s, amount=%g, trade_type=%s", apiURL, orderID, amount, tradeType))

	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Use a simple HTTP client without proxy to avoid SSRF protection blocking localhost
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call BEpusdt API: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read BEpusdt response: %v", err)
	}

	common.SysLog(fmt.Sprintf("BEpusdt response: http_status=%d, body=%s", resp.StatusCode, string(respBody)))

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("BEpusdt HTTP error: status=%d, body=%s", resp.StatusCode, string(respBody))
	}

	var result BEpusdtOrderResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse BEpusdt response: %v, body: %s", err, string(respBody))
	}

	if result.StatusCode != 200 {
		return nil, fmt.Errorf("BEpusdt API error (status_code=%d): %s", result.StatusCode, result.Message)
	}

	// Fix payment_url: BEpusdt may return /pay/checkout-counter/ but the actual cashier is at /pay/cashier/
	if result.Data.PaymentUrl != "" {
		result.Data.PaymentUrl = strings.Replace(result.Data.PaymentUrl, "/pay/checkout-counter/", "/pay/cashier/", 1)
	}

	return &result, nil
}

// GenerateUsdtOrderID generates a unique order ID with "usdt-" prefix.
// Total length ≤ 32 characters, alphanumeric characters only after the prefix.
func GenerateUsdtOrderID() string {
	const prefix = "usdt-"
	const maxLen = 32
	const charset = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

	randLen := maxLen - len(prefix)
	b := make([]byte, randLen)
	for i := range b {
		num := make([]byte, 1)
		_, _ = rand.Read(num)
		b[i] = charset[int(num[0])%len(charset)]
	}
	return prefix + string(b)
}

// CheckBEpusdtOrderStatus queries the BEpusdt API for the order status.
// Returns the status code: 1=waiting, 2=success, 3=expired.
func CheckBEpusdtOrderStatus(tradeID string) (int, error) {
	apiURL := strings.TrimRight(setting.UsdtApiUrl, "/") + "/api/v1/order/query-transaction/" + tradeID

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to call BEpusdt check-status API: %v", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read BEpusdt response: %v", err)
	}

	var result BEpusdtCheckStatusResponse
	if err := common.Unmarshal(respBody, &result); err != nil {
		return 0, fmt.Errorf("failed to parse BEpusdt check-status response: %v", err)
	}

	if result.StatusCode != 200 {
		return 0, fmt.Errorf("BEpusdt check-status error: %s", result.Message)
	}

	return result.Data.Status, nil
}

// StartUsdtOrderExpiryTask starts a background goroutine that checks for expired
// pending USDT orders every 60 seconds. If BEpusdt reports the order as paid (status=2),
// it completes the recharge; otherwise it marks the order as expired.
func StartUsdtOrderExpiryTask() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			processExpiredUsdtOrders()
		}
	}()
	common.SysLog("USDT order expiry task started")
}

func processExpiredUsdtOrders() {
	now := common.GetTimestamp()
	orders, err := model.GetPendingExpiredUsdtOrders(now)
	if err != nil {
		common.SysError("failed to get pending expired USDT orders: " + err.Error())
		return
	}

	for _, order := range orders {
		// Check actual status from BEpusdt if we have a trade ID
		if order.UsdtTradeId != "" {
			status, err := CheckBEpusdtOrderStatus(order.UsdtTradeId)
			if err != nil {
				common.SysError(fmt.Sprintf("failed to check BEpusdt order status for trade_id %s: %v", order.UsdtTradeId, err))
			} else if status == 2 {
				// BEpusdt says paid — complete the recharge
				if err := model.RechargeUsdt(order.TradeNo); err != nil {
					common.SysError(fmt.Sprintf("failed to recharge USDT order %s: %v", order.TradeNo, err))
				} else {
					common.SysLog(fmt.Sprintf("USDT order %s recharged via expiry task (BEpusdt status=2)", order.TradeNo))
				}
				continue
			}
		}

		// Mark as expired
		order.Status = common.TopUpStatusExpired
		if err := order.Update(); err != nil {
			common.SysError(fmt.Sprintf("failed to mark USDT order %s as expired: %v", order.TradeNo, err))
		} else {
			common.SysLog(fmt.Sprintf("USDT order %s marked as expired", order.TradeNo))
		}
	}
}

// ValidateUsdtApiUrl validates that the URL starts with http:// or https:// and is well-formed.
func ValidateUsdtApiUrl(rawURL string) bool {
	if rawURL == "" {
		return false
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		return false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	// Must have a valid host
	return u.Host != ""
}
