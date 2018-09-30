package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

type SliceBlkHasheseReq struct {
	timeslice uint64
	hashes    []common.Hash
}
