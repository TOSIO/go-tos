package transaction

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/log"
	"math/big"

	"github.com/TOSIO/go-tos/params"

	"github.com/TOSIO/go-tos/devbase/event"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

var (
	currentAccountNonce uint64 = 0
	emptyC                     = make(chan struct{}, 1)
	//balance                    = make(map[common.Address]big.Int)
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

func txBlockConstruction(pool core.BlockPoolI, event *event.TypeMux, txRequestInfo *TransInfo) (*types.TxBlock, error) {
	if txRequestInfo.From != crypto.PubkeyToAddress(txRequestInfo.PrivateKey.PublicKey) {
		return nil, fmt.Errorf("PrivateKey err")
	}

	//1. set header
	var txBlockI types.Block
	txBlock := new(types.TxBlock)
	txBlockI = txBlock
	txBlock.Header = types.BlockHeader{
		types.BlockTypeTx,
		utils.GetTimeStamp(),
		txRequestInfo.GasPrice,
		txRequestInfo.GasLimit,
	}

	//2. links
	for len(txBlock.Links) == 0 {
		txBlock.Links = pool.SelectUnverifiedBlock(params.MaxLinksNum)
		/* 	unverifyReq := &core.GetUnverifyBlocksEvent{Hashes: txBlock.Links, Done: make(chan struct{})}
		event.Post(unverifyReq)
		<-unverifyReq.Done
		for _, hash := range unverifyReq.Hashes {
			txBlock.Links = append(txBlock.Links, hash)
		} */
		//copy(txBlock.Links, unverifyReq.Hashes)
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

func Transaction(pool core.BlockPoolI, event *event.TypeMux, txRequestInfo *TransInfo) (common.Hash, error) {
	TxBlock, err := txBlockConstruction(pool, event, txRequestInfo)
	if err != nil {
		return TxBlock.GetHash(), err
	}

	log.Debug("block construction success", "hash", TxBlock.GetHash().String(), "block", TxBlock)
	err = pool.EnQueue(TxBlock)
	if err != nil {
		return TxBlock.GetHash(), err
	}
	log.Debug("Added to the block pool successfully")
	return TxBlock.GetHash(), nil
}
