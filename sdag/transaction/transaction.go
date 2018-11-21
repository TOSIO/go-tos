package transaction

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/log"
	"math/big"
	"sync"

	"github.com/TOSIO/go-tos/params"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type Transaction struct {
	currentAccountNonce uint64
	lock                sync.Mutex
	pool                core.BlockPoolI
}

func New(pool core.BlockPoolI) *Transaction {
	return &Transaction{
		pool: pool,
	}
}

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

func (transaction *Transaction) txBlockConstruction(txRequestInfo *TransInfo) (*types.TxBlock, error) {
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
		txBlock.Links = transaction.pool.SelectUnverifiedBlock(params.MaxLinksNum)
		/* 	unverifyReq := &core.GetUnverifyBlocksEvent{Hashes: txBlock.Links, Done: make(chan struct{})}
		event.Post(unverifyReq)
		<-unverifyReq.Done
		for _, hash := range unverifyReq.Hashes {
			txBlock.Links = append(txBlock.Links, hash)
		} */
		//copy(txBlock.Links, unverifyReq.Hashes)
	}

	//3. accoutnonce
	transaction.lock.Lock()
	txBlock.AccountNonce = transaction.currentAccountNonce
	transaction.currentAccountNonce++
	transaction.lock.Unlock()

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

type BlockConstructionParameter struct {
	Links []common.Hash
	Nonce uint64
}

func (transaction *Transaction) GetBlockConstructionParameter() *BlockConstructionParameter {
	var Links []common.Hash
	Links = transaction.pool.SelectUnverifiedBlock(params.MaxLinksNum)
	for len(Links) == 0 {
		Links = transaction.pool.SelectUnverifiedBlock(params.MaxLinksNum)
	}
	transaction.lock.Lock()
	AccountNonce := transaction.currentAccountNonce
	transaction.currentAccountNonce++
	transaction.lock.Unlock()
	return &BlockConstructionParameter{
		Links: Links,
		Nonce: AccountNonce,
	}
}

func (transaction *Transaction) TransactionSendToSDAG(txRequestInfo *TransInfo) (common.Hash, error) {
	TxBlock, err := transaction.txBlockConstruction(txRequestInfo)
	if err != nil {
		return TxBlock.GetHash(), err
	}

	log.Debug("block construction success", "hash", TxBlock.GetHash().String(), "block", TxBlock)
	err = transaction.pool.EnQueue(TxBlock)
	if err != nil {
		return TxBlock.GetHash(), err
	}
	log.Debug("Added to the block pool successfully")
	return TxBlock.GetHash(), nil
}
