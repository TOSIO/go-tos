package storage

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type blockStorage struct {
	RLP            []byte
	status         types.BlockStatus
	confirmList    []common.Hash
	cumulativeDiff string
}

func GetBlock(hash common.Hash) (interface{}, error) {
	var data interface{}
	data = hash
	return data, nil
}

func PutBlock(hash common.Hash, blockRLP []byte) error {
	return nil
}
