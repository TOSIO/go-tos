package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/protocol"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type SyncMode int

const (
	HeavySync SyncMode = iota
	LightSync
)

/*
type RespPacketI interface {
	NodeId() string
	Items() int
}
*/
type BlockStorageI interface {
	// 根据指定的时间片获取对应的所有区块hash
	GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error)

	// 根据指定的hash集合返回对应的区块（RLP流）
	GetBlocks(hashes []common.Hash) ([][]byte, error)

	GetBlock(hash common.Hash) types.Block

	GetBlocksDiffSet(timeslice uint64, all []common.Hash) ([]common.Hash, error)
}

type SynchroniserI interface {
	Start() error
	Stop()
	//SyncHeavy() error
	//SyncLight() error
	//SendBlock(blk interface{}) error

	RequestBlock(hash common.Hash) error
	Broadcast(hash common.Hash) error
	MarkAnnounced(hash common.Hash, nodeID string)

	DeliverLastTimeSliceResp(id string, timeslice uint64) error
	DeliverBlockHashesResp(id string, ts uint64, hashes []common.Hash) error
	DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error

	DeliverNewBlockResp(id string, data [][]byte) error

	DeliverSYNCBlockRequest(id string, beginPoint *protocol.TimesliceIndex) error
	DeliverSYNCBlockResponse(id string, response *protocol.SYNCBlockResponse) error
	DeliverSYNCBlockACKResponse(id string, response *protocol.SYNCBlockResponseACK) error

	//DeliverBlockHashesResp(id string, resp *SliceBlkHashesResp) error
	//DeliverBlockHashesResp(id string, resp RespPacketI) error
}
