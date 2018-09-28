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
	RequestBlockHashBySlice(slice int64) error
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
	DeliverLastTimeSliceResp(id string, timeslice int64) error
}
