package transaction

import (
	"container/list"
	"crypto/ecdsa"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/opentracing/opentracing-go/log"
	"math/big"
)

var (
	UnverifiedTransactionList *list.List
	MaxConfirm                = 11
)

const (
	TxBlockType    = 1
	MinerBlockType = 2
)

func init() {
	UnverifiedTransactionList = list.New()
}

type ReceiverInfo struct {
	To     common.Address
	Amount *big.Int
}
type transInfo struct {
	From       common.Address
	Receiver   []ReceiverInfo
	GasPrice   *big.Int //tls
	GasLimit   uint64   //gas max value
	PrivateKey *ecdsa.PrivateKey
}

func txBlockConstruction(txRequestInfo *transInfo) (*types.TxBlock, error) {
	if txRequestInfo.From != crypto.PubkeyToAddress(txRequestInfo.PrivateKey.PublicKey) {
		return nil, fmt.Errorf("PrivateKey err")
	}

	//1. set header
	var itxBlock types.Block
	txBlock := new(types.TxBlock)
	itxBlock = txBlock
	txBlock.Header = types.BlockHeader{
		TxBlockType,
		utils.GetTOSTimeStamp(),
		txRequestInfo.GasPrice,
		txRequestInfo.GasLimit,
	}

	//2. links
	Confirm(txBlock.Links)

	//3. accoutnonce
	txBlock.AccountNonce = 100

	//4. txout
	for _, v := range txRequestInfo.Receiver {
		txBlock.Outs = append(txBlock.Outs, types.TxOut{Receiver: v.To, Amount: v.Amount})
	}

	//5. vm code
	txBlock.Payload = []byte{0x0, 0x3b}

	//6. sign
	itxBlock.Sign(txRequestInfo.PrivateKey)

	addUnverifiedTransactionList(itxBlock.GetHash())
	return txBlock, nil
}

func addUnverifiedTransactionList(v interface{}) {
	UnverifiedTransactionList.PushFront(v)
}

func PopUnverifiedTransactionList() (interface{}, error) {
	if UnverifiedTransactionList.Len() > 0 {
		return UnverifiedTransactionList.Remove(UnverifiedTransactionList.Front()), nil
	} else {
		return nil, fmt.Errorf("the list is empty")
	}
}

func Confirm(links []common.Hash) {
	listLen := UnverifiedTransactionList.Len()
	for i := 0; i < listLen && i < MaxConfirm; i++ {
		ihash, err := PopUnverifiedTransactionList()
		if err != nil {
			log.Error(err)
		}
		hash, ok := ihash.(common.Hash)
		if !ok {
			log.Error(fmt.Errorf("hash.(common.Hash)"))
		}
		links = append(links, hash)
	}
}
