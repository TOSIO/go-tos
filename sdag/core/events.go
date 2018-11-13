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

type AnnounceEvent struct {
	Hash common.Hash
}

type NetworkNewBlocksEvent struct {
	Blocks []types.Block
}

type LocalNewBlocksEvent struct {
	Blocks []types.Block
}

type IsolateResponseEvent struct {
	Blocks []types.Block
}

type SYNCResponseEvent struct {
	Blocks []types.Block
}

type RelayBlocksEvent struct {
	Blocks []types.Block
}

type GetBlocksEvent struct {
	Hashes []common.Hash
	//IsOrphan bool
}

type BlockReq struct {
	Hash     common.Hash
	Isolated bool
}

type ErrorGetBlocksEvent struct {
	Requests []BlockReq
	Err      error
}

type NewSYNCTask struct {
	NodeID string
	//FirstMBTimeslice    uint64
	LastTempMBTimeslice uint64
	LastMainBlockNum    uint64
	LastCumulatedDiff   big.Int
}

type SYNCStatusEvent struct {
	Progress          SYNCProgress `json:"progress"`
	BeginTS           uint64       `json:"begin_timeslice"`
	EndTS             uint64       `json:"end_timeslice"`
	CurTS             uint64       `json:"cur_timeslice"`
	Index             uint         `json:"cur_timeslice_index"`
	AccumulateSYNCNum uint64       `json:"cumulated_sync_block_num"`

	BeginTime time.Time `json:"begin_time"`
	EndTime   time.Time `json:"end_time"`

	//TriedOrigin []string
	CurOrigin string `json:"datasource"`
	Err       error  `json:"error"`
}

type GetUnverifyBlocksEvent struct {
	Hashes []common.Hash
	Done   chan struct{}
}
