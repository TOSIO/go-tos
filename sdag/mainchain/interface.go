package mainchain

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

type MainChainI interface {
	// 返回最近一次临时主块所在的时间片
	GetLastTempMainBlkSlice() uint64
	GetPervTail() (common.Hash, *big.Int)
	ComputeCumulativeDiff(toBeAddedBlock types.Block) error
}
