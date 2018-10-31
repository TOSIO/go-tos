package sdag

import "github.com/TOSIO/go-tos/devbase/common"

//sdag的参数配置在此定义
type Config struct {
	NetworkId       uint64 // Network ID to use for selecting peers to connect to
	DatabaseCache   int
	DatabaseHandles int            `toml:"-"`
	Tosbase         common.Address `toml:",omitempty"`
	Mining          bool
}

var DefaultConfig = Config{
	NetworkId:     1,
	DatabaseCache: 768,
	Mining:        false,
}
