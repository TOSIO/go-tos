package sdag

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/vm"
)

// Config define sdag elements
type Config struct {
	NetworkId       uint64 // Network ID to use for selecting peers to connect to
	DatabaseCache   int
	DatabaseHandles int            `toml:"-"`
	Tosbase         common.Address `toml:",omitempty"`
	Mining          bool
	VMConfig        vm.Config
	MessageQueue 	bool
}

var DefaultConfig = Config{
	NetworkId:     1,
	DatabaseCache: 768,
	Mining:        false,
	MessageQueue:  false,
}
