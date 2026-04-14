# 需求文档：USDT 充值（基于 BEpusdt 支付网关）

## 简介

将现有的兑换码充值系统扩展为支持 USDT 充值。系统通过对接外部 BEpusdt（Better Easy Payment USDT）支付网关服务来实现 USDT 收款。用户发起充值后，new-api 后端调用 BEpusdt API 创建支付订单，获取收款地址和 USDT 金额，然后将用户跳转到 BEpusdt 收银台页面完成支付。支付成功后，BEpusdt 通过异步回调通知 new-api 后端完成到账确认和额度增加。

## 术语表

- **System**：new-api 后端服务（Go + Gin）
- **Frontend**：new-api 前端应用（React + Semi Design）
- **BEpusdt**：外部独立部署的加密货币支付网关服务，负责区块链扫描、汇率转换和到账检测
- **BEpusdt_API**：BEpusdt 提供的 HTTP API 接口，用于创建订单、查询状态等
- **API_Auth_Token**：BEpusdt 分配的 API 认证令牌，用于签名计算
- **Checkout_Page**：BEpusdt 提供的独立收银台页面，展示收款地址、金额和倒计时
- **USDT**：Tether 稳定币，锚定美元的加密货币
- **TRC-20**：基于 TRON 区块链的代币标准
- **TopUp_Order**：充值订单记录，存储在 top_ups 表中
- **Trade_ID**：BEpusdt 返回的平台交易号，用于查询订单状态和访问收银台
- **Quota**：用户在系统中的可用额度
- **Callback_Notification**：BEpusdt 支付成功后发送到 notify_url 的 HTTP POST 回调
- **Signature**：基于 MD5 的请求/回调签名，用于验证数据完整性
- **Admin_Dashboard**：管理员后台管理界面

## 需求

### 需求 1：BEpusdt 支付网关配置管理

**用户故事：** 作为管理员，我希望能在后台配置 BEpusdt 支付网关参数，以便控制 USDT 充值功能的开启和行为。

#### 验收标准

1. THE Admin_Dashboard SHALL 在支付设置页面提供 USDT 充值配置区域，包含以下可配置项：启用开关、BEpusdt_API 地址（如 http://your-server:8000）、API_Auth_Token、法币货币代码（默认 "cny"）、区块链网络（默认 "tron"）、代币符号（默认 "usdt"）、USDT 对额度的兑换比率、最低充值金额（法币）、订单超时时间（分钟，默认 15）
2. WHEN 管理员保存 USDT 充值配置时，THE System SHALL 验证 BEpusdt_API 地址为合法的 HTTP 或 HTTPS URL 格式
3. WHEN 管理员启用 USDT 充值且 BEpusdt_API 地址或 API_Auth_Token 为空时，THE System SHALL 拒绝保存并返回错误提示"请先配置 BEpusdt API 地址和认证令牌"
4. THE System SHALL 将 USDT 充值配置持久化到 options 表中，配置项键名以 "Usdt" 为前缀

### 需求 2：用户创建 USDT 充值订单

**用户故事：** 作为用户，我希望能创建 USDT 充值订单并跳转到支付页面，以便通过 USDT 转账完成充值。

#### 验收标准

1. WHEN 用户在充值页面选择 USDT 支付方式并输入充值金额（法币）时，THE System SHALL 生成唯一商户订单号（格式为 "usdt-" 前缀加随机字符串，最大 32 字符），并调用 BEpusdt_API 的 POST /payments/epusdt/v1/order/create-transaction 接口创建支付订单，请求参数包含 order_id、amount（法币金额）、notify_url（回调地址）、redirect_url（支付成功跳转地址）、currency、token、network 和 signature
2. WHEN BEpusdt_API 返回创建成功响应时，THE System SHALL 创建一条状态为 "pending" 的 TopUp_Order，记录用户 ID、法币充值金额、BEpusdt 返回的 actual_amount（USDT 金额）、对应额度、商户订单号、Trade_ID、收款地址（receive_address）、过期时间（expiration_time）和 payment_url
3. WHEN TopUp_Order 创建成功时，THE System SHALL 将 BEpusdt 返回的 payment_url（收银台页面地址）返回给前端
4. WHEN 用户输入的充值金额低于管理员配置的最低充值金额时，THE System SHALL 拒绝创建订单并返回错误提示
5. IF BEpusdt_API 调用失败或返回错误响应，THEN THE System SHALL 返回充值订单创建失败的错误提示，并记录错误日志
6. THE System SHALL 按照 BEpusdt 签名算法生成请求签名：收集所有非空参数（排除 signature），按 key ASCII 升序排列，拼接为 key=value&key=value 格式，末尾追加 API_Auth_Token，计算小写 MD5

### 需求 3：BEpusdt 支付回调处理

**用户故事：** 作为用户，我希望系统能在我完成 USDT 转账后自动确认到账并增加额度，以便无需手动确认。

#### 验收标准

1. THE System SHALL 提供 POST /api/topup/usdt/callback 接口，用于接收 BEpusdt 的 Callback_Notification
2. WHEN 收到 Callback_Notification 时，THE System SHALL 使用 API_Auth_Token 和相同的签名算法验证回调数据中的 Signature 字段，确保回调来源合法
3. IF Callback_Notification 的 Signature 验证失败，THEN THE System SHALL 返回 HTTP 403 状态码并记录安全告警日志
4. WHEN Callback_Notification 的 Signature 验证通过且 status 为 2（支付成功）时，THE System SHALL 根据 order_id 查找对应的 TopUp_Order，在数据库事务中将订单状态更新为 "success"，并按照配置的兑换比率增加用户 Quota
5. THE System SHALL 记录每笔已处理的 Trade_ID，避免同一笔回调被重复处理（幂等性保证）
6. WHEN 回调处理成功时，THE System SHALL 返回 HTTP 200 状态码和纯文本 "ok" 作为响应体
7. IF 回调中的 order_id 在 TopUp_Order 中不存在，THEN THE System SHALL 返回 HTTP 200 状态码和纯文本 "ok"（避免 BEpusdt 重复发送），并记录告警日志

### 需求 4：订单过期处理

**用户故事：** 作为系统运维人员，我希望超时未支付的订单能被自动标记为过期，以便保持订单数据的准确性。

#### 验收标准

1. THE System SHALL 启动后台定时任务，定期（每 60 秒）扫描状态为 "pending" 且已超过 expiration_time 的 TopUp_Order
2. WHEN 检测到过期的 TopUp_Order 时，THE System SHALL 将订单状态更新为 "expired"
3. THE System SHALL 支持通过 BEpusdt_API 的 GET /pay/check-status/:trade_id 接口查询订单实际状态，当本地订单为 "pending" 但 BEpusdt 返回状态为 2（支付成功）时，THE System SHALL 按照支付成功逻辑处理该订单

### 需求 5：充值页面 USDT 支付方式展示

**用户故事：** 作为用户，我希望在充值页面看到 USDT 支付选项，以便选择使用 USDT 进行充值。

#### 验收标准

1. WHILE USDT 充值功能已启用，THE Frontend SHALL 在充值页面的支付方式列表中展示 "USDT (TRC-20)" 选项
2. WHEN 用户选择 USDT 支付方式时，THE Frontend SHALL 展示法币充值金额输入框，允许用户输入希望充值的法币金额
3. WHEN 用户确认创建 USDT 充值订单后，THE Frontend SHALL 在新窗口中打开 BEpusdt 的 Checkout_Page（payment_url），用户在 BEpusdt 收银台页面完成支付操作
4. WHILE 用户等待支付完成时，THE Frontend SHALL 每 10 秒通过 System 查询订单状态，当检测到订单状态变为 "success" 时自动展示充值成功提示
5. WHEN USDT 充值功能未启用时，THE Frontend SHALL 不展示 USDT 支付选项
6. WHEN 订单状态变为 "expired" 时，THE Frontend SHALL 展示订单已过期提示，并引导用户重新创建订单

### 需求 6：USDT 充值订单管理

**用户故事：** 作为管理员，我希望能查看和管理所有 USDT 充值订单，以便监控充值情况和处理异常。

#### 验收标准

1. THE Admin_Dashboard SHALL 在充值记录页面展示所有 TopUp_Order，包含商户订单号、Trade_ID、用户 ID、法币充值金额、USDT 实际金额（actual_amount）、对应额度、支付方式、状态、创建时间和完成时间
2. WHEN 管理员查看充值记录时，THE Admin_Dashboard SHALL 支持按商户订单号、Trade_ID 和用户 ID 搜索
3. WHEN 管理员对状态为 "pending" 或 "expired" 的 USDT 订单执行手动补单操作时，THE System SHALL 在数据库事务中将订单状态更新为 "success" 并增加用户 Quota
4. THE System SHALL 为每次充值成功（自动回调或手动补单）记录操作日志，日志内容包含充值方式、法币金额、USDT 金额、商户订单号和 Trade_ID

### 需求 7：兑换码充值功能保留

**用户故事：** 作为管理员，我希望兑换码充值功能继续可用，以便在特殊场景下（如促销活动）仍可使用兑换码。

#### 验收标准

1. THE System SHALL 保留现有的兑换码创建、管理和兑换功能，接口和行为不变
2. THE Frontend SHALL 在充值页面继续展示兑换码输入区域
3. WHEN 用户输入有效兑换码并提交时，THE System SHALL 按照现有逻辑验证兑换码并增加用户 Quota

### 需求 8：充值信息接口扩展

**用户故事：** 作为前端开发者，我希望充值信息接口返回 USDT 充值的启用状态和配置，以便前端正确展示支付选项。

#### 验收标准

1. THE System SHALL 在 GET /api/user/topup/info 接口的响应中包含 "enable_usdt_topup" 布尔字段，表示 USDT 充值是否已启用
2. WHILE USDT 充值已启用，THE System SHALL 在充值信息接口响应中包含 "usdt_min_topup" 字段（最低充值金额，法币）和 "usdt_currency" 字段（法币货币代码）
3. THE System SHALL 将 USDT 支付方式自动添加到 "pay_methods" 列表中，类型标识为 "usdt"

### 需求 9：回调接口安全防护

**用户故事：** 作为系统运维人员，我希望回调接口具备安全防护能力，以便防止伪造回调攻击。

#### 验收标准

1. THE System SHALL 对回调接口实施速率限制，单个 IP 每分钟最多接受 60 次回调请求
2. IF 回调请求的 Content-Type 不是 application/json 或 application/x-www-form-urlencoded，THEN THE System SHALL 返回 HTTP 400 状态码
3. THE System SHALL 在回调处理的每个关键步骤（签名验证、订单匹配、额度增加）记录详细日志，包含请求来源 IP 和 Trade_ID
