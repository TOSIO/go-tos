package storage

import (
	"strconv"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/devbase/common"
)

var (
	count uint64
	countTime uint64
)

func blockLookUpKey(slice uint64) []byte {
	return append([]byte("t"), strconv.AppendUint(nil, slice, 16)...)
}

func blockKey(slice uint64) []byte {
	countNum := nextId()
	return append(blockLookUpKey(slice), countNum...)
}

func nextId() []byte {

	currentT := utils.GetTimeStamp()
	if countTime != currentT {
		countTime = currentT
		count = 0
	} else {
		count += 1
	}

	return strconv.AppendUint(nil, countTime + count, 16)
}

func blockInfoKey(hash common.Hash) []byte {
	return append([]byte("i"), hash[:]...)
}