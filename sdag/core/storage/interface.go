package storage

import (
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

//type Readteration interface {
//	// Get all block hashes according to the specified time slice
//	GetBlockHashByTmSlice(slice uint64) ([]common.Hash, error)
//
//	// Get the block of RLP according to the specified hashes
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
