package types

import (
	"fmt"
	"math/big"
	"crypto/ecdsa"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
	"sync/atomic"
	"io"
)

//交易输出
type TxOut struct {
	receiver common.Address
	Amount *big.Int
}

//交易区块
type TxBlock struct {
	Header BlockHeader
	Links []common.Hash //外部參數
	AccountNonce uint64
	Outs []TxOut
	Payload []byte

	// Signature values
	BlockSign
	sender atomic.Value

	difficulty  *big.Int
	cumulativeDiff *big.Int
	status BlockStatus

	hash atomic.Value
	size atomic.Value
}

//for sign
type TxBlock_T struct {
	Header BlockHeader
	Links []common.Hash // 外部參數
	AccountNonce uint64
	Outs []TxOut
	Payload []byte
}

func (tx *TxBlock) data(withSig bool) (x interface{}) {
	if withSig {
		x = []interface{}{
			tx.Header,
			tx.Links,
			tx.AccountNonce,
			tx.Outs,
			tx.Payload,
		}
	} else {
		x = tx
	}

	return
}

func (tx *TxBlock) GetRlp() []byte {
	enc, _ := rlp.EncodeToBytes(tx.data(true))
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


func (tx *TxBlock) GetDiff()  *big.Int {
	if tx.difficulty != nil {
		return tx.difficulty
	}

	tx.difficulty=utils.CalculateWork(tx.GetHash())
	return tx.difficulty
}

func (tx *TxBlock) GetCumulativeDiff() *big.Int{
	return tx.cumulativeDiff
}

func (tx *TxBlock) SetCumulativeDiff(cumulativeDiff *big.Int) {
	tx.cumulativeDiff = cumulativeDiff
}

func (tx *TxBlock) GetStatus() BlockStatus {
	return tx.status
}

func (tx *TxBlock) SetStatus(status BlockStatus) {
	tx.status = status
}

func (tx *TxBlock) GetSender() (common.Address, error){
	if sender := tx.sender.Load(); sender != nil {
		return sender.(common.Address), nil
	}

	v , err := recoverPlain(rlpHash(tx.data(false)), tx.R, tx.S, tx.V)
	if err == nil {
		tx.sender.Store(v)
	}

	return v, err
}

func (tx *TxBlock) Sign(prv *ecdsa.PrivateKey) (err error) {
	hash :=  rlpHash(tx.data(false))
	return tx.SignByHash(hash[:], prv)
}

func (tx *TxBlock) GetPublicKey(sighash common.Hash,sig []byte) ([]byte, error){
	pub, err := crypto.Ecrecover(sighash[:], sig)
	return pub,err
}

func (tx *TxBlock) VerifySignature(pubkey, hash, signature []byte) bool {
	return crypto.VerifySignature(pubkey, hash, signature)
}

//validate RlpEncoded TxBlock
func (tx *TxBlock) Validation(tx_in []byte)  error {
	//TODO

	/*
	1.区块是 RLP 格式数据，没有多余的后缀字节;
	2.区块的产生时间不小于Dagger元年；
	3.区块的所有输出金额加上费用之和必须小于TOS总金额;
	4.VerifySignature
	*/

	return nil
}


func (tx *TxBlock) GetLinks() []common.Hash {
	return tx.Links
}

func (tx *TxBlock) GetTime() *big.Int  {
	return new(big.Int).Set(tx.Header.Time)
}

// EncodeRLP implements rlp.Encoder
func (tx *TxBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &tx)
}

// DecodeRLP implements rlp.Decoder
func (tx *TxBlock) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	err := s.Decode(tx)
	if err == nil {
		tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}

	return err
}
