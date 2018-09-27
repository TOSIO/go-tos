package storage

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

type BlockStorageI interface {
	// 根据指定的时间片获取对应的所有区块hash
	GetBlockHashByTmSlice(slice int64) ([]common.Hash, error)

	// 根据指定的hash集合返回对应的区块（RLP流）
	GetBlocks([]common.Hash) ([][]byte, error)
}
