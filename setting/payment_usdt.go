package setting

var (
	UsdtEnabled      bool
	UsdtApiUrl       string            // BEpusdt API 地址，如 http://your-server:8000
	UsdtApiAuthToken string            // BEpusdt API 认证令牌
	UsdtCurrency     string  = "cny"   // 法币货币代码
	UsdtNetwork      string  = "tron"  // 区块链网络
	UsdtToken        string  = "usdt"  // 代币符号
	UsdtQuotaPerUnit float64 = 500000  // 1 USDT 对应的额度
	UsdtMinTopUp     float64 = 10      // 最低充值金额（法币）
	UsdtOrderTimeout int     = 15      // 订单超时时间（分钟）
)
