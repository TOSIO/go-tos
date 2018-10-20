package transaction

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/manager"
)

const (
	TxBlockType     = 1
	MinerBlockType  = 2
	defaultGasPrice = 100
	defaultGasLimit = 1 << 32
)

var (
	currentAccountNonce uint64 = 0
	emptyC                     = make(chan struct{}, 1)
)

type ReceiverInfo struct {
	To     common.Address
	Amount *big.Int
}
type TransInfo struct {
	From       common.Address
	Receiver   []ReceiverInfo
	GasPrice   *big.Int //tls
	GasLimit   uint64   //gas max value
	PrivateKey *ecdsa.PrivateKey
}

func txBlockConstruction(txRequestInfo *TransInfo) (*types.TxBlock, error) {
	if txRequestInfo.From != crypto.PubkeyToAddress(txRequestInfo.PrivateKey.PublicKey) {
		return nil, fmt.Errorf("PrivateKey err")
	}

	//1. set header
	var txBlockI types.Block
	txBlock := new(types.TxBlock)
	txBlockI = txBlock
	txBlock.Header = types.BlockHeader{
		TxBlockType,
		utils.GetTimeStamp(),
		txRequestInfo.GasPrice,
		txRequestInfo.GasLimit,
	}

	//2. links
	for len(txBlock.Links) == 0 {
		txBlock.Links = manager.SelectUnverifiedBlock(txBlock.Links)
	}

	//3. accoutnonce
	emptyC <- struct{}{}
	txBlock.AccountNonce = currentAccountNonce
	currentAccountNonce++
	<-emptyC

	//4. txout
	for _, v := range txRequestInfo.Receiver {
		txBlock.Outs = append(txBlock.Outs, types.TxOut{Receiver: v.To, Amount: v.Amount})
	}

	//5. vm code
	txBlock.Payload = []byte{0x0, 0x3b}

	//6. sign
	err := txBlockI.Sign(txRequestInfo.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("Sign error")
	}

	return txBlock, nil
}

func Transaction(txRequestInfo *TransInfo) error {
	TxBlock, err := txBlockConstruction(txRequestInfo)
	if err != nil {
		return err
	}
	err = manager.SyncAddBlock(TxBlock)
	if err != nil {
		return err
	}
	return nil
}
