package types

import (
	"crypto/ecdsa"
	"math/big"
	"sync/atomic"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
)

type GenesisBlock struct {
	Header   BlockHeader
	RootHash common.Hash

	mutableInfo MutableInfo

	hash atomic.Value
	size atomic.Value
}

func (gb *GenesisBlock) data() (x interface{}) {
	x = []interface{}{
		gb.Header,
		gb.RootHash,
	}

	return x
}

func (gb *GenesisBlock) GetRlp() []byte {
	enc, _ := rlp.EncodeToBytes(gb.data())
	return enc
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (gb *GenesisBlock) GetHash() common.Hash {
	if hash := gb.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(gb.data())
	gb.hash.Store(v)
	return v
}

func (gb *GenesisBlock) GetDiff() *big.Int {
	if gb.mutableInfo.Difficulty != nil {
		return gb.mutableInfo.Difficulty
	}

	gb.mutableInfo.Difficulty = utils.CalculateWork(rlpHash(gb.data()))
	return gb.mutableInfo.Difficulty
}

func (gb *GenesisBlock) GetCumulativeDiff() *big.Int {
	return gb.mutableInfo.CumulativeDiff
}

func (gb *GenesisBlock) SetCumulativeDiff(cumulativeDiff *big.Int) {
	gb.mutableInfo.CumulativeDiff = cumulativeDiff
}

func (gb *GenesisBlock) GetStatus() BlockStatus {
	return gb.mutableInfo.Status
}

func (gb *GenesisBlock) SetStatus(status BlockStatus) {
	gb.mutableInfo.Status = status
}

// GetSender relate sign
func (gb *GenesisBlock) GetSender() (common.Address, error) {
	return common.Address{}, nil
}

func (gb *GenesisBlock) Sign(prv *ecdsa.PrivateKey) error {
	return nil
}

func (gb *GenesisBlock) GetLinks() []common.Hash {
	return []common.Hash{}
}

func (gb *GenesisBlock) GetTime() uint64 {
	return gb.Header.Time
}

func (gb *GenesisBlock) UnRlp(mbRLP []byte) (*GenesisBlock, error) {

	newGb := new(GenesisBlock)

	if err := rlp.DecodeBytes(mbRLP, newGb); err != nil {
		return nil, err
	}

	return newGb, nil
}

// Validation RlpEncoded TxBlock
func (gb *GenesisBlock) Validation() error {
	return nil
}

func (gb *GenesisBlock) GetMutableInfo() *MutableInfo {
	return &gb.mutableInfo
}

func (gb *GenesisBlock) SetMutableInfo(mutableInfo *MutableInfo) {
	gb.mutableInfo = *mutableInfo
}

// SetMaxLink is block interface
func (gb *GenesisBlock) SetMaxLink(MaxLink common.Hash) {
	gb.mutableInfo.MaxLinkHash = MaxLink
}

func (gb *GenesisBlock) GetMaxLink() common.Hash {
	return gb.mutableInfo.MaxLinkHash
}

func (gb *GenesisBlock) GetType() BlockType {
	return gb.Header.Type
}

func (gb *GenesisBlock) GetGasPrice() *big.Int {
	return gb.Header.GasPrice
}

func (gb *GenesisBlock) GetGasLimit() uint64 {
	return gb.Header.GasLimit
}

func (self *GenesisBlock) AsMessage() (Message, error) {
	return Message{}, nil
}
