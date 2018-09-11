package _type

import (
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
)


//交易输出
type TxOut struct {
	receiver common.Address
	Amount *big.Int
}

//交易区块
type TxBlock struct {
	Header BlockHeader
	Links []common.Hash
	AccountNonce uint64
	Outs []TxOut
	Payload []byte

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}