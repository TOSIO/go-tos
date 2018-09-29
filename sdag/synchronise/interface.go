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
	RequestBlockHashBySlice(slice uint64) error
	RequestBlockData(hashes []common.Hash) error
	RequestLastMainSlice() error
}

type PeerSetI interface {
	SelectRandomPeer() (PeerI, error)
}

type RespPacketI interface {
	NodeId() string
	Items() int
}

type SynchroniserI interface {
	SyncHeavy() error
	SyncLight() error
	RequestBlock(blk interface{}) (interface{}, error)
	SendBlock(blk interface{}) error
	DeliverLastTimeSliceResp(id string, timeslice uint64) error
	//DeliverBlockHashesResp(id string, resp *SliceBlkHashesResp) error
	//DeliverBlockHashesResp(id string, ts uint64, hash []common.Hash) error
	DeliverBlockHashesResp(id string, resp RespPacketI) error

	DeliverBlockDatasResp(id string, resp RespPacketI) error
}
