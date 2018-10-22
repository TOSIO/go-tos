package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

const (
	NETWORK_CONNECTED = iota
	NETWORK_CLOSED
)

type NewBlocksEvent struct {
	Blocks []types.Block
}

type RelayBlocksEvent struct {
	Blocks []types.Block
}

type GetBlocksEvent struct {
	Hashes []common.Hash
}

type GetUnverifyBlocksEvent struct {
	Hashes []common.Hash
	Done   chan struct{}
}
