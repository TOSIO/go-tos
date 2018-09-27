package sync

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

type SyncI interface {
	syncHeavy() error
	syncLight() error
	requestBlock(blk interface{}) (interface{}, error)
	sendBlock(blk interface{}) error
}
