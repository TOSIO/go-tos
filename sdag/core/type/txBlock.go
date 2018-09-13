package _type

import (
	"io"
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"

	"crypto/ecdsa"

	"fmt"

	"github.com/TOSIO/go-tos/devbase/crypto"

	"github.com/TOSIO/go-tos/devbase/rlp"
)

type Signer interface {
	SignatureValues(tx *TxBlock, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *TxBlock) common.Hash
}

//交易输出
type TxOut struct {
	receiver common.Address
	Amount   *big.Int
}

//交易区块
type TxBlock struct {
	Header       BlockHeader
	Links        []common.Hash
	AccountNonce uint64
	Outs         []TxOut
	Payload      []byte

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func (tx *TxBlock) Hash() []common.Hash {
	return tx.Links
}
func (tx *TxBlock) Diff() *big.Int {
	return new(big.Int).Set(tx.Header.Difficulty)
}
func (tx *TxBlock) Time() *big.Int {
	return new(big.Int).Set(tx.Header.Time)
}

func (tx *TxBlock) Sender() (b []byte) {
	fmt.Println("hello")
	return []byte("hello")
}

func (tx *TxBlock) Sign() (b []byte) {
	return []byte("hello")
}
func (tx *TxBlock) RlpEncode() (b []byte) {
	return []byte("hello")
}

func (tx *TxBlock) RlpDecode() (b []byte) {
	return []byte("hello")
}
func (tx *TxBlock) Validation() error {
	return nil
}

/*
func SignTx(tx *Transaction, s Signer, prv *ecdsa.PrivateKey) (*Transaction, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}
*/

func SignTx(tx *TxBlock, s Signer, prv *ecdsa.PrivateKey) (*TxBlock, error) {
	h := s.Hash(tx)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}
func (tx *TxBlock) WithSignature(signer Signer, sig []byte) (*TxBlock, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &TxBlock{R: r, S: s, V: v}
	//	cpy.data.R, cpy.data.S, cpy.data.V = r, s, v
	return cpy, nil
}

func SignatureValues(tx *TxBlock, sig []byte) (r, s, v *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	r = new(big.Int).SetBytes(sig[:32])
	s = new(big.Int).SetBytes(sig[32:64])
	v = new(big.Int).SetBytes([]byte{sig[64] + 27})
	return r, s, v, nil
}

//crypto.Sign(hash, key.PrivateKey)

// EncodeRLP implements rlp.Encoder
func (tx *TxBlock) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, tx)
}

// DecodeRLP implements rlp.Decoder
func (tx *TxBlock) DecodeRLP(s *rlp.Stream) error {
	//	_, size, _ := s.Kind()
	err := s.Decode(tx)
	if err == nil {
		//tx.size.Store(common.StorageSize(rlp.ListSize(size)))
	}
	return err
}

// block interface
