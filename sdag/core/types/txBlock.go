package types

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"math/big"
	"github.com/TOSIO/go-tos/devbase/crypto/sha3"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"crypto/ecdsa"
	"fmt"
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


/*
type Signer interface {
	SignatureValues(tx *TxBlock, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *TxBlock) common.Hash
}
*/
type Signer interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *TxBlock) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *TxBlock, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *TxBlock) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(Signer) bool
}
/*
func (s EIP155Signer) Equal(s2 Signer) bool {
	eip155, ok := s2.(EIP155Signer)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}
*/

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}


// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (tx1 *TxBlock) Hash(tx *TxBlock) common.Hash {
	return rlpHash([]interface{}{
		tx.Header,
		tx.Links,
		tx.AccountNonce,
		tx.Outs,
		tx.Payload,
	})
}

func (tx1 *TxBlock) Sender(tx *TxBlock) (common.Address, error) {
	return recoverPlain(tx.Hash(tx), tx.R, tx.S, tx.V)
}

func (tx1 *TxBlock) GetPublicKey(sighash common.Hash,sig []byte) (*ecdsa.PublicKey, error){
	pub, err := crypto.Ecrecover(sighash[:], sig)
	fmt.Println(len(pub))
	//	return crypto.ToECDSAPub(pub),err
	UnmarshalPubkey(pub)
}

func (tx1 *TxBlock) VerifySignature(pubkey, hash, signature []byte) bool {
	return crypto.VerifySignature(pubkey, hash, signature)

}
/*
func (tx1 *TxBlock) Sender(tx *TxBlock) (common.Address, error) {
	return recoverPlain(tx.Hash(tx), tx.R, tx.S, tx.V)
}
*/

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

//交易输出
type TxOut struct {
	receiver common.Address
	Amount *big.Int
}

//1.sign
//2.hash


//交易区块
type TxBlock struct {
	Header BlockHeader
	Links []common.Hash
	AccountNonce uint64
	Outs []TxOut
	Payload []byte

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

	difficulty  *big.Int
	hash common.Hash

}
type TxBlock_T struct {
	Header BlockHeader
	Links []common.Hash // 外部參數
	AccountNonce uint64
	Outs []TxOut
	Payload []byte
}

/*
func (tx *TxBlock) Header() *Header { 
	
	return CopyHeader(tx.tx.Header) 
}
*/

/*
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
/*
func (tx *TxBlock) Hash() []common.Hash {
	return tx.Links
}


func (tx *TxBlock) Diff()  *big.Int {
	return new(big.Int).Set(tx.Header.Difficulty)
}

*/
func (tx *TxBlock) Time() *big.Int  {
	return new(big.Int).Set(tx.Header.Time)
}



func (tx *TxBlock) Sign()(b []byte) {
	return []byte("hello")
}
func (tx *TxBlock) RlpEncode()(b []byte) {
	return []byte("hello")
}

func (tx *TxBlock) Validation()  error {
	return nil
}
//key, _ := HexToECDSA(testPrivHex)
func (tx *TxBlock) SignTxBlock(tx1  *TxBlock, prv *ecdsa.PrivateKey) (*TxBlock, error) {
	h := tx.Hash(tx1)
	sig, err := crypto.Sign(h[:], prv)
	if err != nil {
		return nil, err
	}
	return tx1.WithSignature(sig)
}
func (tx *TxBlock) GetSign(tx1  *TxBlock, prv *ecdsa.PrivateKey) ([]byte, error) {
	h := tx.Hash(tx1)
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
	tx.R=r
	tx.S=s
	tx.V=v
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
	b,err :=rlp.EncodeToBytes(&val)
	return b,err
}

// DecodeRLP implements rlp.Decoder
func (tx TxBlock) DecodeRLP(b []byte,val interface{}) error {
	return rlp.DecodeBytes(b,&val)
}


