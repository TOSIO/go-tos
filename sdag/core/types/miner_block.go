package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
	"math/big"
	"sync/atomic"
)

type BlockNonce [8]byte

//挖矿区块
type MinerBlock struct {
	Header BlockHeader
	Links  []common.Hash
	Miner  common.Address
	Nonce  BlockNonce

	// Signature values
	BlockSign

	//status BlockStatusParam

	sender atomic.Value

	mutableInfo MutableInfo

	hash atomic.Value
	size atomic.Value
}

func (mb *MinerBlock) data(withSig bool) (x interface{}) {
	if withSig {
		x = mb
	} else {
		x = []interface{}{
			mb.Header,
			mb.Links,
			mb.Miner,
			mb.Nonce,
		}
	}

	return
}

func (mb *MinerBlock) GetRlp() []byte {
	enc, _ := rlp.EncodeToBytes(mb.data(true))
	return enc
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (mb *MinerBlock) GetHash() common.Hash {
	if hash := mb.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(mb.data(true))
	mb.hash.Store(v)
	return v
}

func (mb *MinerBlock) GetDiff() *big.Int {
	if mb.mutableInfo.Difficulty != nil {
		return mb.mutableInfo.Difficulty
	}

	mb.mutableInfo.Difficulty = utils.CalculateWork(rlpHash(mb.data(false)))
	return mb.mutableInfo.Difficulty
}

func (mb *MinerBlock) GetCumulativeDiff() *big.Int {
	return mb.mutableInfo.CumulativeDiff
}

func (mb *MinerBlock) SetCumulativeDiff(cumulativeDiff *big.Int) {
	mb.mutableInfo.CumulativeDiff = cumulativeDiff
}

func (mb *MinerBlock) GetStatus() BlockStatus {
	return mb.mutableInfo.Status
}

func (mb *MinerBlock) SetStatus(status BlockStatus) {
	mb.mutableInfo.Status = status
}

//relate sign
func (mb *MinerBlock) GetSender() (common.Address, error) {
	if sender := mb.sender.Load(); sender != nil {
		return sender.(common.Address), nil
	}

	v, err := recoverPlain(rlpHash(mb.data(false)), mb.R, mb.S, mb.V)
	if err == nil {
		mb.sender.Store(v)
	}

	if v != mb.Miner {
		return common.Address{}, errors.New("Signature conflicts with address.")
	}

	return v, err
}

func (mb *MinerBlock) Sign(prv *ecdsa.PrivateKey) error {
	hash := rlpHash(mb.data(false))
	return mb.SignByHash(hash[:], prv)
}

func (mb *MinerBlock) GetLinks() []common.Hash {
	return mb.Links
}

func (mb *MinerBlock) GetTime() uint64 {
	return mb.Header.Time
}

func (mb *MinerBlock) UnRlp(mbRLP []byte) (*MinerBlock, error) {

	newMb := new(MinerBlock)

	if err := rlp.DecodeBytes(mbRLP, newMb); err != nil {
		return nil, err
	}

	if err := newMb.Validation(); err != nil {
		return nil, err
	}

	return newMb, nil
}

//validate RlpEncoded TxBlock
func (mb *MinerBlock) Validation() error {
	//TODO

	/*
		2.区块的产生时间不小于Dagger元年；
		3.区块的所有输出金额加上费用之和必须小于TOS总金额;
		4.VerifySignature
	*/

	//2
	if mb.Header.Time < GenesisTime {
		return fmt.Errorf("block time no greater than Genesis time")
	} else if mb.Header.Time > utils.GetTimeStamp() {
		return fmt.Errorf("block time no less than current time")
	}

	//3

	//4
	if _, err := mb.GetSender(); err != nil {
		return err
	}

	return nil
}

func (mb *MinerBlock) GetMutableInfo() *MutableInfo {
	return &mb.mutableInfo
}

// block interface
