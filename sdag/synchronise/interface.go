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
	// Get all corresponding block hashes according to the specified time slice
	GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error)

	// Get the RLP block information according to the specified hashes
	GetBlocks(hashes []common.Hash) ([][]byte, error)

	GetBlock(hash common.Hash) types.Block

	GetBlocksDiffSet(timeslice uint64, all []common.Hash) ([]common.Hash, error)

	GetMainBlock(number uint64) (common.Hash, error)
}

type SynchroniserI interface {
	Start() error
	Stop()
	//SyncHeavy() error
	//SyncLight() error
	//SendBlock(blk interface{}) error
	Clear(nodeID string)

	RequestBlock(hash common.Hash) error
	RequestIsolatedBlock(hash common.Hash) error

	Broadcast(hash common.Hash) error
	MarkAnnounced(hash common.Hash, nodeID string)

	ExceedAnnounceLimit(node string) bool

	DeliverLastTimeSliceResp(id string, timeslice uint64) error
	DeliverBlockHashesResp(id string, ts uint64, hashes []common.Hash) error
	DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error

	DeliverNewBlockResp(id string, data [][]byte) error

	DeliverSYNCBlockRequest(id string, beginPoint *protocol.TimesliceIndex) error
	DeliverSYNCBlockResponse(id string, response *protocol.SYNCBlockResponse) error
	DeliverSYNCBlockACKResponse(id string, response *protocol.SYNCBlockResponseACK) error

	DeliverLocatorResponse(id string, response []protocol.MainChainSample) error

	//DeliverBlockHashesResp(id string, resp *SliceBlkHashesResp) error
	//DeliverBlockHashesResp(id string, resp RespPacketI) error
}
