package types

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto/sha3"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/rlp"

	"math/big"
	"errors"
	"fmt"
	"crypto/ecdsa"
)

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}


func recoverPlain(sigHash common.Hash, R, S, Vb *big.Int) (common.Address, error) {
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
	pub, err := crypto.Ecrecover(sigHash[:], sig)
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

type BlockSign struct {
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func (bs *BlockSign) WithSignature(sig []byte) error {
	r, s, v, err := bs.SignatureValues(sig)
	if err != nil {
		return err
	}
	bs.R,bs.S,bs.V=r,s,v
	return nil
}

func (bs *BlockSign)SignatureValues(sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

//签名
func (bs *BlockSign) SignByHash(hash []byte, prv *ecdsa.PrivateKey) error {
	sig, err := crypto.Sign(hash, prv)
	if err != nil {
		return err
	}
	return bs.WithSignature(sig)
}
//
//func (tx *TxBlock) GetSign(prv *ecdsa.PrivateKey) ([]byte, error) {
//	h := tx.Hash(false)
//	sig, err := crypto.Sign(h[:], prv)
//	if err != nil {
//		return nil, err
//	}
//	return sig, err
//}

