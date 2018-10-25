package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type BlockPoolI interface {
	SelectUnverifiedBlock(num int) []common.Hash
	EnQueue(block types.Block) error
	//GetUserBlockStatus(hash common.Hash) string
	//GetBlockInfo(hash common.Hash) string
}

type Peer interface {
	NodeID() string

	SetIdle(idle bool)
	RequestBlockHashBySlice(slice uint64) error
	RequestBlocksBySlice(timeslice uint64, hashes []common.Hash) error
	RequestBlock(hash common.Hash) error
	//RequestBlocks(hashes []common.Hash) error
	RequestLastMainSlice() error
	SendNewBlockHash(hash common.Hash) error
}

type PeerSet interface {
	RandomSelectIdlePeer() (Peer, error)
	Peers() map[string]Peer
	FindPeer(string) Peer
}

type Response interface {
	NodeID() string
	ItemCount() int
}
