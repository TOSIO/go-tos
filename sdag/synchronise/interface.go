package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

type SyncMode int

const (
	HeavySync SyncMode = iota
	LightSync
)

type PeerI interface {
	NodeID() string
	SetIdle(idle bool)
	RequestBlockHashBySlice(slice uint64) error
	RequestBlockData(timeslice uint64, hashes []common.Hash) error
	RequestLastMainSlice() error
}

type PeerSetI interface {
	RandomSelectIdlePeer() (PeerI, error)
}

type RespPacketI interface {
	NodeId() string
	Items() int
}

type BlockStorageI interface {
	// 根据指定的时间片获取对应的所有区块hash
	GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error)

	// 根据指定的hash集合返回对应的区块（RLP流）
	GetBlocks([]common.Hash) ([][]byte, error)

	GetBlocksDiffSet(timeslice uint64, all []common.Hash) ([]common.Hash, error)

	AddBlock(block []byte) error
}

type SynchroniserI interface {
	SyncHeavy() error
	SyncLight() error
	RequestBlock(blk interface{}) (interface{}, error)
	SendBlock(blk interface{}) error

	DeliverLastTimeSliceResp(id string, timeslice uint64) error
	DeliverBlockHashesResp(id string, ts uint64, hashes []common.Hash) error
	DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error

	//DeliverBlockHashesResp(id string, resp *SliceBlkHashesResp) error
	//DeliverBlockHashesResp(id string, resp RespPacketI) error
}
