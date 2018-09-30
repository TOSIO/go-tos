package types

import (
	"crypto/ecdsa"
	"math/big"

	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
	"sync/atomic"
)

//交易输出
type TxOut struct {
	Receiver common.Address
	Amount   *big.Int //tls
}

//交易区块
type TxBlock struct {
	Header       BlockHeader
	Links        []common.Hash // 链接的区块hash
	AccountNonce uint64        // 100
	Outs         []TxOut       //
	Payload      []byte        // vm code  0x0

	// Signature values
	BlockSign
	sender atomic.Value

	mutableInfo MutableInfo

	hash atomic.Value
	size atomic.Value
}

func (tx *TxBlock) data(withSig bool) (x interface{}) {
	if withSig {
		x = tx
	} else {
		x = []interface{}{
			tx.Header,
			tx.Links,
			tx.AccountNonce,
			tx.Outs,
			tx.Payload,
		}
	}

	return
}

func (tx *TxBlock) GetRlp() []byte {
	enc, err := rlp.EncodeToBytes(tx.data(true))
	if err != nil {
		fmt.Println("err: ", err)
	}
	return enc
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (tx *TxBlock) GetHash() common.Hash {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	v := rlpHash(tx.data(true))
	tx.hash.Store(v)
	return v
}

func (tx *TxBlock) GetDiff() *big.Int {
	if tx.mutableInfo.Difficulty != nil {
		return tx.mutableInfo.Difficulty
	}

	tx.mutableInfo.Difficulty = utils.CalculateWork(tx.GetHash())
	return tx.mutableInfo.Difficulty
}

func (tx *TxBlock) GetCumulativeDiff() *big.Int {
	return tx.mutableInfo.CumulativeDiff
}

func (tx *TxBlock) SetCumulativeDiff(cumulativeDiff *big.Int) {
	tx.mutableInfo.CumulativeDiff = cumulativeDiff
}

func (tx *TxBlock) GetStatus() BlockStatus {
	return tx.mutableInfo.Status
}

func (tx *TxBlock) SetStatus(status BlockStatus) {
	tx.mutableInfo.Status = status
}

//relate sign
func (tx *TxBlock) GetSender() (common.Address, error) {
	if sender := tx.sender.Load(); sender != nil {
		return sender.(common.Address), nil
	}

	v, err := recoverPlain(rlpHash(tx.data(false)), tx.R, tx.S, tx.V)
	if err == nil {
		tx.sender.Store(v)
	}

	return v, err
}

func (tx *TxBlock) Sign(prv *ecdsa.PrivateKey) error {
	hash := rlpHash(tx.data(false))
	return tx.SignByHash(hash[:], prv)
}

func (tx *TxBlock) GetLinks() []common.Hash {
	return tx.Links
}

func (tx *TxBlock) GetTime() uint64 {
	return tx.Header.Time
}

func (tx *TxBlock) UnRlp(txRLP []byte) (*TxBlock, error) {

	newTx := new(TxBlock)

	if err := rlp.DecodeBytes(txRLP, newTx); err != nil {
		return nil, err
	}

	if err := newTx.Validation(); err != nil {
		return nil, err
	}

	return newTx, nil
}

//validate RlpEncoded TxBlock
func (tx *TxBlock) Validation() error {
	//TODO

	/*
		2.区块的产生时间不小于Dagger元年；
		3.区块的所有输出金额加上费用之和必须小于TOS总金额;
		4.VerifySignature
	*/

	//2
	if tx.Header.Time < GenesisTime {
		return fmt.Errorf("block time no greater than Genesis time")
	}

	//3
	amount := big.NewInt(0)
	for _, out := range tx.Outs {
		amount.Add(amount, out.Amount)
	}

	if !(amount.Cmp(GlobalTosTotal) < 0) {
		return fmt.Errorf("The amount is not less than the GlobalTosTotal")
	}

	//4
	if _, err := tx.GetSender(); err != nil {
		return err
	}

	return nil
}

func (tx *TxBlock) GetMutableInfo() *MutableInfo {
	return &tx.mutableInfo
}
