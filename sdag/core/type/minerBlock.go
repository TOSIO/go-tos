package _type

import (
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	_ "github.com/TOSIO/go-tos/devbase/rlp"
)

type BlockNonce [8]byte

//挖矿区块
type MinerBlock struct {
	Header     BlockHeader
	Links      []common.Hash
	Sender     common.Address
	Nonce      BlockNonce
	Difficulty big.Int
	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func (m *MinerBlock) Hash() []common.Hash {
	return m.Links
}
func (m *MinerBlock) Diff() *big.Int {
	return new(big.Int).Set(m.Header.Difficulty)
}
func (m *MinerBlock) Time() *big.Int {
	return new(big.Int).Set(m.Header.Time)
}

func (m *MinerBlock) GetSender() (b []byte) {
	return []byte("hello")
}

func (m *MinerBlock) Sign() (b []byte) {
	return []byte("hello")
}
func (m *MinerBlock) RlpEncode() (b []byte) {
	return []byte("hello")
}

func (m *MinerBlock) RlpDecode() (b []byte) {
	return []byte("hello")
}
func (m *MinerBlock) Validation() error {
	return nil
}

// block interface
