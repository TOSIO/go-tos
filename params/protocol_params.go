package params

const (
	TimePeriod   = 60000 // 60000 毫秒
	MaxLinksNum  = 4     //最大链接次数
	ConfirmBlock = 32    //最大确定个数
)

const (
	OneTos                     = 1e18 //1 tos = 1e18 tls
	DefaultGasPrice            = 100
	DefaultGasLimit            = 1 << 32
	TransferTransactionGasUsed = 21000     // 21000 tls
	MiningGasUsed              = 21000     // 21000 tls
	InitialRewardMiner         = 50 * 1e18 //50 tos
	HalfLifeRewardMiner        = 2000000   //Half down every 2,000,000 main blocks
	ConfirmUserRewardRate      = 0.2
	ConfirmMinerRewardRate     = 1 - ConfirmUserRewardRate
)
