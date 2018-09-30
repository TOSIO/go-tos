package storage

import (
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

//type Readteration interface {
//	// 根据指定的时间片获取对应的所有区块hash
//	GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error)
//
//	// 根据指定的hash集合返回对应的区块（RLP流）
//	GetBlocks([]common.Hash) ([][]byte, error)
//}


// DatabaseReader wraps the Has and Get method of a backing data store.
type Reader interface {
	Has(key []byte) (bool, error)
	Get(key []byte) ([]byte, error)
}

// DatabaseWriter wraps the Put method of a backing data store.
type Writer interface {
	Put(key []byte, value []byte) error
}

// DatabaseDeleter wraps the Delete method of a backing data store.
type Deleter interface {
	Delete(key []byte) error
}

type ReaderWrite interface {
	Reader
	Writer
}

type ReadIteration interface {
	Reader
	NewIteratorWithPrefix(prefix []byte) iterator.Iterator
}
