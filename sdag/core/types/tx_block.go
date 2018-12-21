package types

import (
	"crypto/ecdsa"
	"github.com/TOSIO/go-tos/devbase/log"
	"math/big"

	"github.com/TOSIO/go-tos/params"

	"fmt"
	"sync/atomic"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
)

type TxOut struct {
	Receiver common.Address
	Amount   *big.Int //tls
}

type TxBlock struct {
	Header       BlockHeader
	Links        []common.Hash // block link hash
	AccountNonce uint64        // 100
	Outs         []TxOut       //
	Payload      []byte        // vm code  0x0

	// Signature values
	V *big.Int `json:"v" gencodec:"required"`
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`

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
		log.Error("GetRlp" + err.Error())
	}
	return enc
}

// GetHash returns the hash to be signed by the sender.
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

//GetSender relate sign
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

	var err error
	tx.R, tx.S, tx.V, err = (&BlockSign{
		tx.V,
		tx.R,
		tx.S,
	}).SignByHash(hash[:], prv)

	return err
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

//Validation RlpEncoded TxBlock
func (tx *TxBlock) Validation() error {
	if tx.Header.GasPrice.Cmp(params.GasPriceMin) < 0 || tx.Header.GasLimit <= 0 {
		return fmt.Errorf("GasPrice is less than the minimum")
	}

	if tx.Header.Time < GenesisTime {
		return fmt.Errorf("block time no greater than Genesis time")
	} else if tx.Header.Time > utils.GetTimeStamp()+params.TimePeriod/4 {
		return fmt.Errorf("block time no less than current time")
	}

	linksNumber := len(tx.Links)
	if linksNumber < 1 || linksNumber > params.MaxLinksNum {
		return fmt.Errorf("the block linksNumber =%d", linksNumber)
	}

	amount := big.NewInt(0)
	for _, out := range tx.Outs {
		amount.Add(amount, out.Amount)
	}

	if !(amount.Cmp(params.GlobalTosTotal) < 0) {
		return fmt.Errorf("the amount is not less than the GlobalTosTotal")
	}

	var isContract bool
	if len(tx.Payload) > 0 {
		if len(tx.Payload) > params.PayloadLenLimit {
			return fmt.Errorf("payload is too long len(tx.Payload)=%d", len(tx.Payload))
		}
		isContract = true
		if len(tx.Outs) > 1 {
			return fmt.Errorf("contract transaction len(tx.Outs) > 1")
		}
	} else {
		if len(tx.Outs) < 1 || len(tx.Outs) > params.MaxTransNum {
			return fmt.Errorf("len(tx.Outs)=%d number error", len(tx.Outs))
		}
	}

	from, err := tx.GetSender()
	if err != nil {
		return err
	}

	for _, out := range tx.Outs {
		if out.Receiver == from {
			return fmt.Errorf("the receiver is self")
		}
		if !isContract {
			if out.Amount.Sign() <= 0 {
				return fmt.Errorf("the amount=[%s] must be positive", out.Amount.String())
			}
			if tx.Outs[0].Receiver == (common.Address{}) {
				return fmt.Errorf("receiver cannot be empty")
			}
		}
	}

	for i := 0; i < len(tx.Links); i++ {
		for j := i + 1; j < len(tx.Links); j++ {
			if tx.Links[i] == tx.Links[j] {
				return fmt.Errorf("links repeat")
			}
		}
	}

	return nil
}

func (tx *TxBlock) GetMutableInfo() *MutableInfo {
	return &tx.mutableInfo
}

func (tx *TxBlock) SetMutableInfo(mutableInfo *MutableInfo) {
	tx.mutableInfo = *mutableInfo
}

func (tx *TxBlock) GetMaxLink() common.Hash {
	return tx.mutableInfo.MaxLinkHash
}

func (tx *TxBlock) SetMaxLink(MaxLink common.Hash) {
	tx.mutableInfo.MaxLinkHash = MaxLink
}

func (tx *TxBlock) GetType() BlockType {
	return tx.Header.Type
}

func (tx *TxBlock) GetGasPrice() *big.Int {
	return tx.Header.GasPrice
}

func (tx *TxBlock) GetGasLimit() uint64 {
	return tx.Header.GasLimit
}

func (tx *TxBlock) GetPayload() []byte {
	return tx.Payload
}

func (tx *TxBlock) Nonce() uint64 {
	return tx.AccountNonce
}

func (tx *TxBlock) AsMessage() (Message, error) {
	var to *common.Address
	amount := big.NewInt(0)
	if len(tx.Outs) > 0 {
		amount = tx.Outs[0].Amount
		if tx.Outs[0].Receiver != (common.Address{}) {
			to = &tx.Outs[0].Receiver
		}
	}
	msg := Message{
		nonce:      tx.AccountNonce,
		gasLimit:   tx.Header.GasLimit,
		gasPrice:   tx.Header.GasPrice,
		to:         to,
		amount:     amount,
		data:       tx.Payload,
		checkNonce: false,
	}

	var err error
	msg.from, err = tx.GetSender()
	return msg, err

}

func (tx *TxBlock) GetOuts() []TxOut {
	return tx.Outs
}

func (tx *TxBlock) String() string {
	str := fmt.Sprintf("hash:%s\n", tx.GetHash().String())
	str += fmt.Sprintf("head:{\n")
	str += fmt.Sprintf("Type:%d\n", tx.Header.Type)
	str += fmt.Sprintf("Time:%d\n", tx.Header.Time)
	str += fmt.Sprintf("GasPrice:%s\n", tx.Header.GasPrice.String())
	str += fmt.Sprintf("GasLimit:%d\n", tx.Header.GasLimit)
	str += fmt.Sprintf("}\n")
	str += fmt.Sprintf("links:{\n")
	for i, link := range tx.Links {
		str += fmt.Sprintf("link[%d]:%s\n", i, link.String())
	}
	str += fmt.Sprintf("}\n")
	sender, _ := tx.GetSender()
	str += fmt.Sprintf("sender:%s\n", sender.String())
	str += fmt.Sprintf("outs:{\n")
	for i, out := range tx.Outs {
		str += fmt.Sprintf("out[%d]:Amount:%s,Receiver:%s\n", i, out.Amount.String(), out.Receiver.String())
	}
	str += fmt.Sprintf("}\n")
	str += fmt.Sprintf("AccountNonce:%d\n", tx.AccountNonce)
	str += fmt.Sprintf("Payload:%s\n", common.Bytes2Hex(tx.Payload))
	str += fmt.Sprintf("mutableInfo:{\n")
	str += fmt.Sprintf("Status:%b\n", tx.mutableInfo.Status)
	str += fmt.Sprintf("ConfirmItsNumber:%d\n", tx.mutableInfo.ConfirmItsNumber)
	str += fmt.Sprintf("ConfirmItsIndex:%d\n", tx.mutableInfo.ConfirmItsIndex)
	str += fmt.Sprintf("Difficulty:%s\n", tx.mutableInfo.Difficulty.String())
	str += fmt.Sprintf("CumulativeDiff:%s\n", tx.mutableInfo.CumulativeDiff.String())
	str += fmt.Sprintf("MaxLinkHash:%s\n", tx.mutableInfo.MaxLinkHash.String())
	str += fmt.Sprintf("}\n")
	return str
}
