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
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
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

func (mb *MinerBlock) GetSender() (common.Address, error) {
	return recoverPlain(mb.Hash(false), mb.R, mb.S, mb.V)
}

func (mb *MinerBlock) GetPublicKey(sighash common.Hash,sig []byte) ([]byte, error){
	pub, err := crypto.Ecrecover(sighash[:], sig)
	return pub,err
}

func (mb *MinerBlock) VerifySignature(pubkey, hash, signature []byte) bool {
	return crypto.VerifySignature(pubkey, hash, signature) 
	 
}

func (mb *MinerBlock)recoverPlain(sighash common.Hash, R, S, Vb *big.Int) (common.Address, error) {
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

//get cached value

func (mb *MinerBlock) Diff(hash common.Hash)  *big.Int {
	mb.Difficulty=utils.CalculateWork(hash)  
	return mb.Difficulty
}

func (mb *MinerBlock) GetHash_sig(hash common.Hash)  common.Hash {
	return mb.hash_sig
}

func (mb *MinerBlock) GetHash()  common.Hash {
	return mb.hash
}

//get block elements

func (mb *MinerBlock) GetHeader() BlockHeader {
	return mb.Header 
}
func (mb *MinerBlock) GetLinks() []common.Hash {
	return mb.Links 
}
func (mb *MinerBlock) GetAccountNonce() BlockNonce {
	return mb.Nonce 
}


//get blockheader elements

func (mb *MinerBlock) Type() uint32  { 
	return mb.Header.Type 
}
func (mb *MinerBlock) Time() *big.Int  { 
	return new(big.Int).Set(mb.Header.Time) 
}
func (mb *MinerBlock) GasPrice() *big.Int  { 
	return new(big.Int).Set(mb.Header.GasPrice) 
}

func (mb *MinerBlock) GasLimit() uint64  { 
	return mb.Header.GasLimit 
}



//validate RlpEncoded TxBlock
func (mb *MinerBlock) Validation(tx_in []byte)  error {
	/*
	1.区块是 RLP 格式数据，没有多余的后缀字节;
	2.区块的产生时间不小于Dagger元年；
	3.区块的所有输出金额加上费用之和必须小于TOS总金额;
	4.VerifySignature
	*/
	//	DecodeRLP(tx_in)
	
	return nil
}

func (mb *MinerBlock) Sign( prv *ecdsa.PrivateKey) (*MinerBlock, error) {
	h := mb.Hash(false)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return mb.WithSignature(sig)
}
func (mb *MinerBlock) GetSign(prv *ecdsa.PrivateKey) ([]byte, error) {
	h := mb.Hash(false)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return sig, err
}

func (mb *MinerBlock) WithSignature(sig []byte) (*MinerBlock, error) {
	r, s, v, err := mb.SignatureValues(sig)
	if err != nil {
		return nil, err
	}
	mb.R,mb.S,mb.V=r,s,v
	return mb, nil
}


func (mb *MinerBlock) SignatureValues(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}


func (mb *MinerBlock) EncodeRLP(val interface{}) ([]byte,error) {
	b,err :=rlp.EncodeToBytes(val) 
	return b,err
}

// DecodeRLP implements rlp.Decoder
func (mb *MinerBlock) DecodeRLP(b []byte,val interface{}) error {
	return rlp.DecodeBytes(b,val)
}



// block interface
