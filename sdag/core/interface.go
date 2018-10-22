package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type BlockPoolI interface {
	SelectUnverifiedBlock(num int) []common.Hash
	EnQueue(block types.Block) error
}
