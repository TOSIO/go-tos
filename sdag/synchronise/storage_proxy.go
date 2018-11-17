package synchronise

import (
	"errors"
	"github.com/deckarep/golang-set"

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
	hashes, err := storage.ReadBlocksHashByTmSlice(s.db, timeslice)
	if len(hashes) <= 0 || err != nil {
		return all, nil
	}
	//diff := make([]common.Hash,0)
	in := mapset.NewSet()
	local := mapset.NewSet()
	for _, hash := range all {
		in.Add(hash)
	}
	for _, hash := range hashes {
		local.Add(hash)
	}
	diff := in.Difference(local)
	itr := diff.Iterator()
	result := make([]common.Hash, 0)
	for e := range itr.C {
		result = append(result, e.(common.Hash))
	}
	return result, err
}

func (s *StorageProxy) GetBlock(hash common.Hash) types.Block {
	return storage.ReadBlock(s.db, hash)
}

func (s *StorageProxy) GetMainBlock(number uint64) (common.Hash, error) {
	mainblockInfo, err := storage.ReadMainBlock(s.db, number)
	if err != nil {
		return common.Hash{}, err
	}
	mainblockHash := mainblockInfo.Hash
	return mainblockHash, nil
}
