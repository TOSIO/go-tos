package storage

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

//type blockStorage struct {
//	RLP            []byte
//	status         types.BlockStatus
//	confirmList    []common.Hash
//	cumulativeDiff string
//}

func GetBlock(hash common.Hash) (interface{}, error) {
	var data interface{}
	data = hash
	return data, nil
}

func PutBlock(hash common.Hash, blockRLP []byte) error {
	return nil
}

func Update(hash common.Hash, data interface{}, update func(block types.Block, data interface{})) error {
	linkBlockEI, err := GetBlock(hash) //the 'EI' is empty interface logogram
	if err == nil {
		if block, ok := linkBlockEI.(types.Block); ok {
			update(block, data)
		} else {
			log.Error("linkBlockEI assertion failure")
			return fmt.Errorf("linkBlockEI assertion failure")
		}
	} else {
		log.Error("GetBlock fail")
		return fmt.Errorf("GetBlock fail")
	}
	return nil
}
