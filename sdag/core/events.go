package core

import (
	"math/big"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

const (
	NETWORK_CONNECTED = iota
	NETWORK_CLOSED
	SDAGSYNC_SYNCING
	SDAGSYNC_COMPLETED
)

type SYNCProgress int

const (
	SYNC_READY SYNCProgress = iota
	SYNC_SYNCING
	SYNC_ERROR
	SYNC_END
)

var syncStrDICT = map[SYNCProgress]string{
	SYNC_READY:   "SYNC-READY",
	SYNC_SYNCING: "SYNC-SYNCING",
	SYNC_ERROR:   "SYNC-ERROR",
	SYNC_END:     "SYNC-END",
}

func (s *SYNCProgress) String() string {
	return syncStrDICT[SYNCProgress(*s)]
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

type NewSYNCTask struct {
	NodeID              string
	FirstMBTimeslice    uint64
	LastTempMBTimeslice uint64
	LastMainBlockNum    uint64
	LastCumulatedDiff   big.Int
}

type SYNCStatusEvent struct {
	Progress SYNCProgress
	BeginTS  uint64
	EndTS    uint64
	CurTS    uint64

	AccumulateSYNCNum uint64

	BeginTime time.Time
	EndTime   time.Time

	//TriedOrigin []string
	CurOrigin string
	Err       error
}

type GetUnverifyBlocksEvent struct {
	Hashes []common.Hash
	Done   chan struct{}
}
