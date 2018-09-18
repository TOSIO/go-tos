package types

import (
	"fmt"
	"math/big"
	"crypto/ecdsa"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"	
	"github.com/TOSIO/go-tos/devbase/crypto/sha3"
	"errors"
)
var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

/*

type Block interface {
	Hash() common.Hash 		//获取区块hash,包括签名,tx,miner block is the same
	Diff(hash common.Hash) *big.Int	  		//获取区块难度,pow.go,calutae 传入hash(tx:包含签名,miner:不包括签名 )
	Time() uint64				//获取区块时间
	Sender() common.Address     //获取区块发送者，即创建者,从签名获取
	Sign() 
	Links() []common.Address
	RlpEncode()
	RlpDecode()
	Validation() (check data,校验解签名)
	//签名
	//RLP编解码
}

type BlockHeader struct {
	Type uint32
	Time *big.Int
	GasPrice *big.Int
	GasLimit uint64


}

*/

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
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	difficulty  *big.Int
	hash_sig 	common.Hash
	hash	common.Hash
	
}
type TxBlock_T struct {
	Header BlockHeader
	Links []common.Hash // 外部參數
	AccountNonce uint64
	Outs []TxOut
	Payload []byte
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}


// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (tx *TxBlock) Hash(sig bool) common.Hash {
	if sig == false {
	return rlpHash([]interface{}{
		tx.Header,
		tx.Links,
		tx.AccountNonce,
		tx.Outs,
		tx.Payload,
	})
	} else {
	return rlpHash([]interface{}{
		tx.Header,
		tx.Links,
		tx.AccountNonce,
		tx.Outs,
		tx.Payload,
		tx.R,
		tx.S,
		tx.V,
	})
	}
}

func (tx *TxBlock) Sender() (common.Address, error) {
	return recoverPlain(tx.Hash(false), tx.R, tx.S, tx.V)
}

func (tx *TxBlock) GetPublicKey(sighash common.Hash,sig []byte) ([]byte, error){
	pub, err := crypto.Ecrecover(sighash[:], sig)
	return pub,err
}

func (tx *TxBlock) VerifySignature(pubkey, hash, signature []byte) bool {
	return crypto.VerifySignature(pubkey, hash, signature) 
	 
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !crypto.ValidateSignatureValues(V, R, S, false) {
		return common.Address{}, ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the signature
	pub, err := crypto.Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], crypto.Keccak256(pub[1:])[12:])
	return addr, nil
}


/*
func (tx *TxBlock) Header() *Header { 
	
	return CopyHeader(tx.Header) 
}



func (tx *TxBlock) CopyHeader(h *BlockHeader) *BlockHeader {
	cpy := *h
	if cpy.Time = new(big.Int); h.Time != nil {
		cpy.Time.Set(h.Time)
	}
	if cpy.Difficulty = new(big.Int); h.Difficulty != nil {
		cpy.Difficulty.Set(h.Difficulty)
	}
	if cpy.Number = new(big.Int); h.Number != nil {
		cpy.Number.Set(h.Number)
	}
	if len(h.Extra) > 0 {
		cpy.Extra = make([]byte, len(h.Extra))
		copy(cpy.Extra, h.Extra)
	}
	return &cpy
}

*/

//get cached value

func (tx *TxBlock) Diff(hash common.Hash)  *big.Int {
	tx.difficulty=utils.CalculateWork(hash)  
	return tx.difficulty
}

func (tx *TxBlock) GetHash_sig(hash common.Hash)  common.Hash {
	return tx.hash_sig
}

func (tx *TxBlock) GetHash()  common.Hash {
	return tx.hash
}

//get block elements

func (tx *TxBlock) GetHeader() BlockHeader {
	return tx.Header 
}
func (tx *TxBlock) GetLinks() []common.Hash {
	return tx.Links 
}
func (tx *TxBlock) GetAccountNonce() uint64 {
	return tx.AccountNonce 
}
func (tx *TxBlock) GetTxOut() []TxOut {
	return tx.Outs 
}
func (tx *TxBlock) GetPayload() []byte {
	return tx.Payload 
}

//get blockheader elements

func (tx *TxBlock) Type() uint32  { 
	return tx.Header.Type 
}
func (tx *TxBlock) Time() *big.Int  { 
	return new(big.Int).Set(tx.Header.Time) 
}
func (tx *TxBlock) GasPrice() *big.Int  { 
	return new(big.Int).Set(tx.Header.GasPrice) 
}

func (tx *TxBlock) GasLimit() uint64  { 
	return tx.Header.GasLimit 
}



//validate RlpEncoded TxBlock
func (tx *TxBlock) Validation(tx_in []byte)  error {
	/*
	1.区块是 RLP 格式数据，没有多余的后缀字节;
	2.区块的产生时间不小于Dagger元年；
	3.区块的所有输出金额加上费用之和必须小于TOS总金额;
	4.VerifySignature
	*/
	//	DecodeRLP(tx_in)
	
	return nil
}

func (tx *TxBlock) Sign( prv *ecdsa.PrivateKey) (*TxBlock, error) {
	h := tx.Hash(false)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(sig)
}
func (tx *TxBlock) GetSign(prv *ecdsa.PrivateKey) ([]byte, error) {
	h := tx.Hash(false)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return sig, err
}

func (tx *TxBlock) WithSignature(sig []byte) (*TxBlock, error) {
	r, s, v, err := tx.SignatureValues(sig)
	if err != nil {
		return nil, err
	}
	tx.R,tx.S,tx.V=r,s,v
	return tx, nil
}


func (tx *TxBlock) SignatureValues(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}


func (tx TxBlock) EncodeRLP(val interface{}) ([]byte,error) {
	b,err :=rlp.EncodeToBytes(val) 
	return b,err
}

// DecodeRLP implements rlp.Decoder
func (tx TxBlock) DecodeRLP(b []byte,val interface{}) error {
	return rlp.DecodeBytes(b,val)
}


