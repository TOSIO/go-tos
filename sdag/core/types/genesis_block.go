package types

import (
	"crypto/ecdsa"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
	"math/big"
	"sync/atomic"
)

//创世区块
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

//relate sign
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

//validate RlpEncoded TxBlock
func (gb *GenesisBlock) Validation() error {
	return nil
}

func (gb *GenesisBlock) GetMutableInfo() *MutableInfo {
	return &gb.mutableInfo
}

func (gb *GenesisBlock) SetMutableInfo(mutableInfo *MutableInfo) {
	gb.mutableInfo = *mutableInfo
}

// block interface
func (gb *GenesisBlock) SetMaxLinks(MaxLink uint8) {
	gb.mutableInfo.MaxLink = MaxLink
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