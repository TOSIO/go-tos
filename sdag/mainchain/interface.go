package mainchain

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

type MainChainI interface {
	//  return the time slice of the last time temporary block
	GetLastTempMainBlkSlice() uint64
	GetPervTail() (common.Hash, *big.Int)
	GetTail() *types.TailMainBlockInfo
	GetMainTail() *types.TailMainBlockInfo
	ComputeCumulativeDiff(toBeAddedBlock types.Block) (bool, error)
	UpdateTail(block types.Block)
	GetGenesisHash() (common.Hash, error)
	GetNextMain(hash common.Hash) (common.Hash, *types.MutableInfo, error)

	AddLocalAddress(address common.Address)
	LocalBlockNoticeSend(hash common.Hash, address common.Address, action int)
	LocalBlockNoticeChan() chan LocalBlockNotice
}
