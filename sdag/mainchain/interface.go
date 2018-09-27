package mainchain

type MainChainI interface {
	// 返回最近一次临时主块所在的时间片
	GetLastTempMainBlkSlice() int64
}
