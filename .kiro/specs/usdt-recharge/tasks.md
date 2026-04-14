# 实现计划：USDT 充值（基于 BEpusdt 支付网关）

## 概述

在现有 new-api 充值系统上新增 USDT 充值通道，对接 BEpusdt 支付网关。按照分层架构（Setting → Model → Service → Controller → Router → Frontend）逐步实现，每步构建在前一步基础上，确保无孤立代码。

## 任务

- [x] 1. 新增 USDT 支付配置变量和 options 表注册
  - [x] 1.1 创建 `setting/payment_usdt.go`，定义 USDT 配置变量（UsdtEnabled、UsdtApiUrl、UsdtApiAuthToken、UsdtCurrency、UsdtNetwork、UsdtToken、UsdtQuotaPerUnit、UsdtMinTopUp、UsdtOrderTimeout），参照 `setting/payment_waffo.go` 模式
    - _需求: 1.1, 1.4_
  - [x] 1.2 在 `model/option.go` 的 `InitOptionMap()` 中添加所有 Usdt 前缀配置项的默认值注册
    - _需求: 1.4_
  - [x] 1.3 在 `model/option.go` 的 `updateOptionMap()` 中添加所有 Usdt 前缀配置项的运行时同步逻辑（string→变量赋值）
    - _需求: 1.4_

- [x] 2. 扩展 TopUp 数据模型
  - [x] 2.1 在 `model/topup.go` 的 `TopUp` 结构体中新增字段：UsdtTradeId（varchar(255), index）、UsdtActualAmount（varchar(64)）、UsdtAddress（varchar(255)）、PaymentUrl（text）、ExpirationTime（int64），GORM AutoMigrate 自动添加列
    - _需求: 2.2_
  - [x] 2.2 在 `model/topup.go` 中新增 `GetPendingExpiredUsdtOrders(now int64)` 函数，查询 status="pending" 且 expiration_time > 0 且 expiration_time < now 的订单列表
    - _需求: 4.1, 4.2_
  - [x] 2.3 在 `model/topup.go` 中新增 `RechargeUsdt(tradeNo string) error` 函数，在数据库事务中完成订单状态更新和用户额度增加（参照 `RechargeWaffo` 模式，使用 `UsdtQuotaPerUnit` 计算额度）
    - _需求: 3.4, 3.5_

- [x] 3. 实现 USDT 业务逻辑层（service）
  - [x] 3.1 创建 `service/usdt.go`，实现 `ComputeBEpusdtSignature(params map[string]string, authToken string) string`：收集非空非 signature 参数，按 key ASCII 升序排列，拼接 key=value&，末尾追加 authToken，计算小写 MD5
    - _需求: 2.6_
  - [x] 3.2 实现 `VerifyBEpusdtSignature(params map[string]string, signature string, authToken string) bool`：调用 ComputeBEpusdtSignature 并比较结果
    - _需求: 3.2_
  - [x] 3.3 实现 `CreateBEpusdtOrder(orderID string, amount float64, notifyURL string, redirectURL string) (*BEpusdtOrderResponse, error)`：构造请求参数、计算签名、POST 调用 BEpusdt API `/payments/epusdt/v1/order/create-transaction`，解析响应（使用 `common.Marshal`/`common.Unmarshal`）
    - _需求: 2.1, 2.2_
  - [x] 3.4 实现 `GenerateUsdtOrderID() string`：生成 "usdt-" 前缀 + 随机字符串，总长度 ≤ 32 字符
    - _需求: 2.1_
  - [x] 3.5 实现 `CheckBEpusdtOrderStatus(tradeID string) (int, error)`：GET 调用 BEpusdt API `/pay/check-status/:trade_id`，返回订单状态码
    - _需求: 4.3_
  - [x] 3.6 实现 `StartUsdtOrderExpiryTask()`：启动 goroutine，每 60 秒扫描过期 pending 订单，调用 BEpusdt 查询实际状态，若已支付则走充值逻辑，否则标记为 expired
    - _需求: 4.1, 4.2, 4.3_
  - [x] 3.7 实现 `ValidateUsdtApiUrl(url string) bool`：验证 URL 以 http:// 或 https:// 开头且格式合法
    - _需求: 1.2_

- [x] 4. 检查点 - 确保后端基础层编译通过
  - 确保所有代码编译通过，如有问题请询问用户。

- [x] 5. 实现 USDT 控制器层（controller）
  - [x] 5.1 创建 `controller/topup_usdt.go`，实现 `RequestUsdtPay(c *gin.Context)`：校验 UsdtEnabled、解析金额、校验最低充值金额、调用 service 创建 BEpusdt 订单、创建 TopUp 记录（status=pending）、返回 payment_url
    - _需求: 2.1, 2.2, 2.3, 2.4, 2.5_
  - [x] 5.2 实现 `UsdtCallback(c *gin.Context)`：校验 Content-Type、解析回调参数、验证签名（失败返回 403 并记录安全日志含来源 IP）、幂等处理（已成功直接返回 ok）、order_id 不存在返回 ok 并记录告警、status=2 时调用 model.RechargeUsdt 完成充值、返回纯文本 "ok"
    - _需求: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.7, 9.2, 9.3_
  - [x] 5.3 实现 `GetUsdtOrderStatus(c *gin.Context)`：根据 trade_no 查询 TopUp 记录，返回订单状态
    - _需求: 5.4_
  - [x] 5.4 修改 `controller/topup.go` 的 `GetTopUpInfo()` 函数，当 UsdtEnabled 且 UsdtApiUrl 和 UsdtApiAuthToken 非空时，在 data 中添加 `enable_usdt_topup`、`usdt_min_topup`、`usdt_currency` 字段，并在 payMethods 中自动添加 USDT 支付方式
    - _需求: 8.1, 8.2, 8.3_

- [x] 6. 注册路由和启动过期任务
  - [x] 6.1 修改 `router/api-router.go`，在 selfRoute 下注册 `POST /usdt/pay`（CriticalRateLimit）和 `GET /usdt/order/status`，在 apiRouter 下注册 `POST /topup/usdt/callback`（CriticalRateLimit）
    - _需求: 2.1, 3.1, 9.1_
  - [x] 6.2 在应用启动入口（`main.go` 或合适位置）调用 `service.StartUsdtOrderExpiryTask()` 启动过期检查定时任务
    - _需求: 4.1_

- [x] 7. 检查点 - 确保后端完整编译通过并验证路由注册
  - 确保所有代码编译通过，如有问题请询问用户。

- [x] 8. 实现前端 USDT 支付流程
  - [x] 8.1 修改 `web/src/components/topup/RechargeCard.jsx`，当 `enable_usdt_topup` 为 true 时在支付方式列表中添加 "USDT (TRC-20)" 选项卡；选择后展示法币金额输入框；提交后调用 `POST /api/user/usdt/pay`，在新窗口打开返回的 payment_url；启动 10 秒间隔轮询 `GET /api/user/usdt/order/status`，成功时展示充值成功提示，过期时展示过期提示
    - _需求: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_

- [x] 9. 实现管理员 USDT 配置页面
  - [x] 9.1 创建 `web/src/pages/Setting/Payment/SettingsPaymentGatewayUsdt.jsx`，参照 `SettingsPaymentGatewayWaffo.jsx` 模式，包含启用开关、BEpusdt API 地址、API 认证令牌、法币货币代码、区块链网络、代币符号、兑换比率、最低充值金额、订单超时时间等配置项；启用时校验必填项（API 地址和认证令牌）
    - _需求: 1.1, 1.2, 1.3_
  - [x] 9.2 修改 `web/src/pages/Setting/Payment/SettingsPaymentGateway.jsx`，引入 USDT 配置 Tab
    - _需求: 1.1_

- [x] 10. 检查点 - 确保前后端完整功能可用
  - 确保所有代码编译通过，如有问题请询问用户。

- [ ]* 11. 编写属性测试（Property-Based Tests）
  - [ ]* 11.1 在 `service/usdt_test.go` 中编写签名往返一致性属性测试
    - **Property 1: BEpusdt 签名计算与验证的往返一致性**
    - **验证: 需求 2.6, 3.2**
  - [ ]* 11.2 在 `service/usdt_test.go` 中编写订单号格式约束属性测试
    - **Property 2: USDT 订单号格式约束**
    - **验证: 需求 2.1**
  - [ ]* 11.3 在 `service/usdt_test.go` 中编写最低充值金额校验属性测试
    - **Property 3: 最低充值金额校验**
    - **验证: 需求 2.4**
  - [ ]* 11.4 在 `service/usdt_test.go` 中编写额度计算正确性属性测试
    - **Property 4: 额度计算正确性**
    - **验证: 需求 3.4**
  - [ ]* 11.5 在 `service/usdt_test.go` 中编写 URL 格式验证属性测试
    - **Property 7: BEpusdt API URL 格式验证**
    - **验证: 需求 1.2**

- [ ]* 12. 编写单元测试
  - [ ]* 12.1 在 `service/usdt_test.go` 中编写签名计算的已知向量测试（使用 BEpusdt 文档示例数据）
    - _需求: 2.6_
  - [ ]* 12.2 在 `controller/topup_usdt_test.go` 中编写回调处理测试（各种 status 码、签名失败、order_id 不存在、重复回调幂等性）
    - _需求: 3.2, 3.3, 3.5, 3.7, 9.2_
  - [ ]* 12.3 在 `service/usdt_test.go` 中编写配置验证边界条件测试（空 URL、空 Token、启用但未配置）
    - _需求: 1.2, 1.3_

- [x] 13. 最终检查点 - 确保所有测试通过
  - 确保所有测试通过，如有问题请询问用户。

## 备注

- 标记 `*` 的任务为可选，可跳过以加速 MVP 交付
- 每个任务引用了具体的需求编号，确保可追溯性
- 检查点确保增量验证，避免问题累积
- 属性测试验证设计文档中定义的正确性属性
- 单元测试验证具体示例和边界条件
- 所有 JSON 操作必须使用 `common.Marshal`/`common.Unmarshal`（项目规范 Rule 1）
- 数据库操作必须兼容 SQLite/MySQL/PostgreSQL（项目规范 Rule 2）
