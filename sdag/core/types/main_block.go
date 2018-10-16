package types

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"fmt"
)

type MainBlock struct {
	Hash common.Hash  //block hash
	Root common.Hash //status root
}

func (mb *MainBlock)Rlp() []byte {
	rlpByte, err := rlp.EncodeToBytes(mb)
	if err != nil {
		fmt.Println("err: ", err)
	}
	return rlpByte
}

func (mb *MainBlock)UnRlp(rlpByte []byte) (*MainBlock , error) {
	if err := rlp.DecodeBytes(rlpByte, mb); err != nil {
		return nil, err
	}

	return mb, nil
}
