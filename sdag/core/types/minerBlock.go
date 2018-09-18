package types


import (
	"math/big"
	"fmt"
	"crypto/ecdsa"
	"errors"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"	
)

type BlockNonce [8]byte

//挖矿区块
type MinerBlock struct {
	Header BlockHeader
	Links []common.Hash
	Sender common.Address
	Nonce BlockNonce
	Difficulty  *big.Int

	// Signature values
	BlockSign

	hash_sig 	common.Hash
	hash	common.Hash
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (mb *MinerBlock) Hash(sig bool) common.Hash {
	if sig == false {
	return rlpHash([]interface{}{
		mb.Header,
		mb.Links,
		mb.Nonce,
		mb.Difficulty,
	})
	} else {
	return rlpHash([]interface{}{
		mb.Header,
		mb.Links,
		mb.Nonce,
		mb.Difficulty,
		mb.R,
		mb.S,
		mb.V,
	})
	}
}


// block interface
