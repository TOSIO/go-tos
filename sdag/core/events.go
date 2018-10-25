package core

import (
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

const (
	NETWORK_CONNECTED = iota
	NETWORK_CLOSED
)

const (
	SYNC_BEGIN = iota
	SYNC_SYNCING
	SYNC_PAUSE
	SYNC_END
)

var syncStrDICT = map[int]string{
	SYNC_BEGIN:   "SYNC-BEGIN",
	SYNC_SYNCING: "SYNC-SYNCING",
	SYNC_PAUSE:   "SYNC-PUSE",
	SYNC_END:     "SYNC-END",
}

func SyncCodeToString(code int) string {
	return syncStrDICT[code]
}

type NewBlocksEvent struct {
	Blocks []types.Block
}

type RelayBlocksEvent struct {
	Blocks []types.Block
}

type GetBlocksEvent struct {
	Hashes []common.Hash
}

type SYNCStatusEvent struct {
	Progress int
	BeginTS  uint64
	EndTS    uint64
	CurTS    uint64

	AccumulateSYNCNum uint64

	BeginTime time.Time
	EndTime   time.Time

	TriedOrigin []string
	CurOrigin   string
	Err         error
}

type GetUnverifyBlocksEvent struct {
	Hashes []common.Hash
	Done   chan struct{}
}
