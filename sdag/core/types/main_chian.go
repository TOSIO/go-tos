package types

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"math/big"
)

type MainBlockInfo struct {
	Hash common.Hash //block hash
	Root common.Hash //status root
}

type TailMainBlockInfo struct {
	Hash           common.Hash
	CumulativeDiff *big.Int
	Number         uint64
	Time           uint64//
}

func (mb *MainBlockInfo) Rlp() []byte {
	rlpByte, err := rlp.EncodeToBytes(mb)
	if err != nil {
		fmt.Println("err: ", err)
	}
	return rlpByte
}

func (mb *MainBlockInfo) UnRlp(rlpByte []byte) (*MainBlockInfo, error) {
	if err := rlp.DecodeBytes(rlpByte, mb); err != nil {
		return nil, err
	}

	return mb, nil
}

func (tm *TailMainBlockInfo) Rlp() []byte {
	rlpByte, err := rlp.EncodeToBytes(tm)
	if err != nil {
		fmt.Println("err: ", err)
	}
	return rlpByte
}

func (tm *TailMainBlockInfo) UnRlp(rlpByte []byte) (*TailMainBlockInfo, error) {
	if err := rlp.DecodeBytes(rlpByte, tm); err != nil {
		return nil, err
	}

	return tm, nil
}
