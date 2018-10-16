package storage

import (
	"strconv"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/common/hexutil"
)

var (
	count uint64
	countTime uint64
)

func blockLookUpKey(slice uint64) []byte {
	return append([]byte("t"), hexutil.EncodeUnit64ToByte(slice)...)
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

	return hexutil.EncodeUnit64ToByte(countTime + count)
}

func blockInfoKey(hash common.Hash) []byte {
	return append([]byte("i"), hash[:]...)
}

func mainBlockKey(slice uint64) []byte {
	return append([]byte("M"), hexutil.EncodeUnit64ToByte(slice)...)
}
