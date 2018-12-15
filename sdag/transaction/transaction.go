package transaction

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/mainchain"
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
	currentAccountNonce    uint64
	nonceLock              sync.Mutex
	addressTransactionLock sync.RWMutex
	addressTransactionMap  map[common.Address]map[common.Hash]*big.Int
	pool                   core.BlockPoolI
	mainChainI             mainchain.MainChainI
	chainDb                tosdb.Database //Block chain database
	stateDb                state.Database //trie
}

func New(pool core.BlockPoolI, mainChainI mainchain.MainChainI, chainDb tosdb.Database, stateDb state.Database) *Transaction {
	transaction := &Transaction{
		addressTransactionMap: make(map[common.Address]map[common.Hash]*big.Int),
		pool:                  pool,
		mainChainI:            mainChainI,
		chainDb:               chainDb,
		stateDb:               stateDb,
	}

	transaction.ReceiveTransactionAction()
	return transaction
}

type ReceiverInfo struct {
	To     common.Address
	Amount *big.Int
}
type TransInfo struct {
	From       common.Address
	Outs       []types.TxOut
	Payload    string
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
	transaction.nonceLock.Lock()
	txBlock.AccountNonce = transaction.currentAccountNonce
	transaction.currentAccountNonce++
	transaction.nonceLock.Unlock()

	//4. txout
	for _, v := range txRequestInfo.Outs {
		txBlock.Outs = append(txBlock.Outs, types.TxOut{Receiver: v.Receiver, Amount: v.Amount})
	}

	//5. vm code
	txBlock.Payload = common.FromHex(txRequestInfo.Payload)

	//6. sign
	err := txBlockI.Sign(txRequestInfo.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("sign error")
	}

	err = txBlock.Validation()
	return txBlock, err
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
	transaction.nonceLock.Lock()
	AccountNonce := transaction.currentAccountNonce
	transaction.currentAccountNonce++
	transaction.nonceLock.Unlock()
	return &BlockConstructionParameter{
		Links: Links,
		Nonce: AccountNonce,
	}
}

func (transaction *Transaction) BlockTransactionSendToSDAG(txRequestInfo *TransInfo) (common.Hash, error) {
	amount := transaction.calculateAmount(txRequestInfo.Outs)
	ok, err := transaction.transactionCheck(txRequestInfo.From, amount)
	if err != nil {
		return common.Hash{}, err
	}
	if !ok {
		return common.Hash{}, fmt.Errorf("insufficient balance")
	}

	TxBlock, err := transaction.txBlockConstruction(txRequestInfo)
	if err != nil {
		return TxBlock.GetHash(), err
	}

	transaction.transactionRecord(txRequestInfo.From, TxBlock.GetHash(), amount)

	return transaction.transactionSendToSDAG(TxBlock)
}

func (transaction *Transaction) RlpTransactionSendToSDAG(rlpData []byte) (hash common.Hash, err error) {
	Block, err := types.BlockDecode(rlpData)
	if err != nil {
		return common.Hash{}, err
	}
	if Block.GetType() != types.BlockTypeTx {
		return common.Hash{}, errors.New("block type error")
	}

	txBlock, ok := Block.(*types.TxBlock)
	if !ok {
		return common.Hash{}, fmt.Errorf("RlpTransactionSendToSDAG Block.(*types.TxBlock) error")
	}

	amount := transaction.calculateAmount(txBlock.Outs)
	from, err := txBlock.GetSender()
	if err != nil {
		return common.Hash{}, fmt.Errorf("RlpTransactionSendToSDAG GetSender error:" + err.Error())
	}

	ok, err = transaction.transactionCheck(from, amount)
	if err != nil {
		return common.Hash{}, fmt.Errorf("RlpTransactionSendToSDAG transactionCheck error:" + err.Error())
	}
	if !ok {
		return common.Hash{}, fmt.Errorf("insufficient balance")
	}
	transaction.transactionRecord(from, txBlock.GetHash(), amount)

	return transaction.transactionSendToSDAG(Block)
}

func (transaction *Transaction) transactionSendToSDAG(block types.Block) (common.Hash, error) {
	log.Debug("block construction success", "hash", block.GetHash().String(), "block", block)
	err := transaction.pool.EnQueue(block)
	if err != nil {
		return block.GetHash(), err
	}
	log.Debug("Added to the block pool successfully")

	return block.GetHash(), nil
}

func (transaction *Transaction) ReceiveTransactionAction() {
	go func() {
		for transactionAction := range transaction.mainChainI.LocalBlockNoticeChan() {
			if transactionAction.Action == mainchain.ConfirmAction {
				transaction.addressTransactionLock.Lock()
				delete(transaction.addressTransactionMap[transactionAction.Address], transactionAction.Hash)
				transaction.addressTransactionLock.Unlock()
			} else if transactionAction.Action == mainchain.RollBackAction {
				block := storage.ReadBlock(transaction.chainDb, transactionAction.Hash)
				if block == nil {
					log.Error("ReadBlock error", "hash", transactionAction.Hash)
					continue
				}
				if block.GetType() != types.BlockTypeTx {
					continue
				}
				txBlock, ok := block.(*types.TxBlock)
				if !ok {
					log.Error("block.(*types.TxBlock) error", "block", block)
					continue
				}
				amount := transaction.calculateAmount(txBlock.Outs)
				transaction.addressTransactionLock.Lock()
				transaction.addressTransactionMap[transactionAction.Address][transactionAction.Hash] = amount
				transaction.addressTransactionLock.Unlock()
			}
		}
	}()
}

func (transaction *Transaction) transactionCheck(address common.Address, amount *big.Int) (bool, error) {
	balance, err := transaction.GetBalance(address)
	if err != nil {
		return false, fmt.Errorf("GetBalance error:" + err.Error())
	}
	log.Debug("transactionCheck balance", "balance", balance.String(), "address", address.String(), "amount", amount.String())

	if balance.Cmp(amount) < 0 {
		return false, nil
	}
	historyAmountTotal := big.NewInt(0)
	transaction.addressTransactionLock.RLock()
	if _, ok := transaction.addressTransactionMap[address]; ok {
		for _, amount := range transaction.addressTransactionMap[address] {
			historyAmountTotal.Add(historyAmountTotal, amount)
		}
	}
	transaction.addressTransactionLock.RUnlock()
	log.Debug("transactionCheck balance", "balance", balance.String(), "address", address.String(), "amount", amount.String(), "historyAmountTotal", historyAmountTotal.String())
	if new(big.Int).Sub(balance, historyAmountTotal).Cmp(amount) < 0 {
		log.Debug("insufficient balance")
		return false, nil
	}

	return true, nil
}

//维护本地交易记录
func (transaction *Transaction) transactionRecord(address common.Address, hash common.Hash, amount *big.Int) {
	transaction.addressTransactionLock.Lock()
	_, ok := transaction.addressTransactionMap[address]
	if !ok {
		transaction.mainChainI.AddLocalAddress(address)
		TransactionMap := make(map[common.Hash]*big.Int)
		TransactionMap[hash] = amount
		transaction.addressTransactionMap[address] = TransactionMap
	} else {
		transaction.addressTransactionMap[address][hash] = amount
	}
	transaction.addressTransactionLock.Unlock()
}

func (transaction *Transaction) calculateAmount(Outs []types.TxOut) *big.Int {
	amount := big.NewInt(0)
	for _, v := range Outs {
		amount.Add(amount, v.Amount)
	}

	return amount
}

func (transaction *Transaction) GetBalance(address common.Address) (*big.Int, error) {
	tailMainBlockInfo := transaction.mainChainI.GetMainTail()
	mainInfo, err := storage.ReadMainBlock(transaction.chainDb, tailMainBlockInfo.Number)
	if err != nil {
		return nil, err
	}
	stateTrie, err := state.New(mainInfo.Root, transaction.stateDb)
	if err != nil {
		return nil, err
	}
	return stateTrie.GetBalance(address), nil
}
