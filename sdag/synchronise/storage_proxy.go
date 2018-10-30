package synchronise

import (
	"errors"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type StorageProxy struct {
	db *tosdb.LDBDatabase
}

var (
	errTypeConvert = errors.New("Type convert failed.")
)

func NewStorage(db tosdb.Database) (*StorageProxy, error) {
	var ok bool
	var lvldb *tosdb.LDBDatabase

	if lvldb, ok = db.(*tosdb.LDBDatabase); !ok {
		return nil, errTypeConvert
	}

	return &StorageProxy{db: lvldb}, nil
}

// 根据指定的时间片获取对应的所有区块hash
func (s *StorageProxy) GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error) {

	return storage.ReadBlocksHashByTmSlice(s.db, slice)
}

// 根据指定的hash集合返回对应的区块（RLP流）
func (s *StorageProxy) GetBlocks(hashes []common.Hash) ([][]byte, error) {
	return storage.ReadBlocks(s.db, hashes)
}

func (s *StorageProxy) GetBlocksDiffSet(timeslice uint64, all []common.Hash) ([]common.Hash, error) {
	local, err := storage.ReadBlocksHashByTmSlice(s.db, timeslice)
	return local, err
}

func (s *StorageProxy) GetBlock(hash common.Hash) types.Block {
	return storage.ReadBlock(s.db, hash)
}
