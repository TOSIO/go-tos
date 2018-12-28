// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package sdag

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common/hexutil"
	"github.com/TOSIO/go-tos/devbase/common/math"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/core/vm"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"time"

	"github.com/TOSIO/go-tos/devbase/statistics"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
	"github.com/pborman/uuid"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/log"

	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/sdag/transaction"
)

var (
	statisticsObj statistics.Statistics
	emptyC        = make(chan struct{}, 1)
)

// PublicEthereumAPI provides an API to access Ethereum full node-related
// information.
type PublicSdagAPI struct {
	s *Sdag
}

// NewPublicEthereumAPI creates a new Ethereum protocol API for full nodes.
func NewPublicSdagAPI(s *Sdag) *PublicSdagAPI {
	return &PublicSdagAPI{s}
}

func (api *PublicSdagAPI) DoRequest(data string) string {
	log.Trace("func PublicSdagAPI.DoRequest | receive request,", "param", data)
	return ""
}
func (api *PublicSdagAPI) Status() (interface{}, error) {
	/* 	data, err := json.Marshal(api.s.Status())
	   	if err != nil {
	   		return ""
	   	} else {
	   		return string(data)
	   	} */
	return api.s.Status(), nil
}

type accountInfo struct {
	Address    string
	PrivateKey string
	Passphrase string
}

type TransactionInfo struct {
	From     accountInfo
	To       string
	Amount   string
	Payload  string
	GasPrice string
	GasLimit uint64
}

type MainBlockInfo struct {
	Number uint64
}

type BlockHashInfo struct {
	BlockHash string
}

type WalletAddress struct {
	WalletAddr string
}

type RpcMinerInfo struct {
	Address  string
	Password string
}

type RpcGenerKeyStore struct {
	Password string
}

type BaseDataParam struct {
	to   string
	data []byte
}

func (api *PublicSdagAPI) GetBlockInfo(blockHashInfo *BlockHashInfo) (interface{}, error) {

	db := api.s.chainDb
	blockInfo, err := api.s.queryBlockInfo.GetBlockInfo(db, common.HexToHash(blockHashInfo.BlockHash))
	if err != nil {
		return "", err
	}
	Receipt, err := storage.ReadReceiptInfo(db, common.HexToHash(blockHashInfo.BlockHash))
	if err == nil {
		//获取GasUsed
		blockInfo.GasUsed = Receipt.GasUsed
		blockInfo.ActualTxFee = new(big.Int).Mul(new(big.Int).SetUint64(Receipt.GasUsed), blockInfo.GasPrice).String()
	}

	return blockInfo, nil

}

func (api *PublicSdagAPI) GetMainBlockInfo(mainBlockTime *MainBlockInfo) (interface{}, error) {
	db := api.s.chainDb
	tempQueryMainBlockInfo, err := api.s.queryBlockInfo.GetMainBlockInfo(db, mainBlockTime.Number)
	if err != nil {
		return "", err
	}

	return tempQueryMainBlockInfo, nil

}

func (api *PublicSdagAPI) GetFinalMainBlockInfo() (interface{}, error) {

	mainBlockInfo, err := api.s.queryBlockInfo.GetFinalMainBlockInfo(api.s.chainDb)
	if err != nil {
		return "", err
	}

	return mainBlockInfo, nil
}

type TransactionParameter struct {
	Links    []common.Hash `json:"links"`
	Nonce    uint64
	Gasprice string `json:"gasprice"`
	GasLimit uint64 `json:"gaslimit"`
}

// GetBaseData get gasPrice  gasLimimt links、nonce
func (api *PublicSdagAPI) GetBaseData() interface{} {
	//if base ==nil{
	//	return fmt.Errorf("to address invalid or data is null")
	//}

	paramtr := api.s.transaction.GetBlockConstructionParameter()
	param := TransactionParameter{
		Links:    paramtr.Links,
		Nonce:    paramtr.Nonce,
		Gasprice: params.DefaultGasPrice.String(),
		GasLimit: params.DefaultGasLimit,
	}
	return param

}
func (api *PublicSdagAPI) Transaction(transactionInfo *TransactionInfo) (interface{}, error) {
	log.Debug("RPC receives Transaction", "receives transactionInfo", transactionInfo)
	var txRequestInfo transaction.TransInfo

	txRequestInfo.From = common.HexToAddress(transactionInfo.From.Address)
	if txRequestInfo.From == (common.Address{}) {
		return nil, fmt.Errorf("from address invalid")
	}

	to := common.HexToAddress(transactionInfo.To)
	if to == (common.Address{}) && len(transactionInfo.Payload) == 0 {
		return nil, fmt.Errorf("to address invalid")
	}

	Amount := new(big.Int)
	var ok bool
	_, ok = Amount.SetString(transactionInfo.Amount, 10)
	if !ok {
		log.Error("Amount is invalid", "Amount", transactionInfo.Amount)
		return nil, fmt.Errorf("amount is invalid")
	}
	if Amount.Sign() < 0 {
		log.Error("The amount must be positive", "Amount", transactionInfo.Amount)
		return nil, fmt.Errorf("the amount=[%s] must be positive", transactionInfo.Amount)
	}
	if len(transactionInfo.GasPrice) == 0 {
		txRequestInfo.GasPrice = params.DefaultGasPrice
	} else {
		txRequestInfo.GasPrice, ok = new(big.Int).SetString(transactionInfo.GasPrice, 10)
		if !ok {
			log.Error("GasPrice is invalid", "GasPrice", transactionInfo.GasPrice)
			return nil, fmt.Errorf("GasPrice is invalid")
		}
	}

	var err error
	txRequestInfo.GasLimit = transactionInfo.GasLimit
	if txRequestInfo.GasLimit == 0 {
		txRequestInfo.GasLimit = params.DefaultGasLimit
	}

	txRequestInfo.Outs = append(txRequestInfo.Outs, types.TxOut{to, Amount})

	txRequestInfo.Payload = transactionInfo.Payload

	if len(transactionInfo.From.PrivateKey) == 0 {
		if len(transactionInfo.From.Passphrase) == 0 {
			log.Error("Passphrase/PrivateKey invalid")
			return nil, fmt.Errorf("Passphrase/PrivateKey invalid")
		}
		txRequestInfo.PrivateKey, err = api.s.accountManager.FindPrivateKey(txRequestInfo.From, transactionInfo.From.Passphrase)
		if err != nil {
			log.Error("PrivateKey invalid:" + err.Error())
			return nil, fmt.Errorf("PrivateKey invalid:%s", err.Error())
		}
	} else {
		txRequestInfo.PrivateKey, err = crypto.HexToECDSA(transactionInfo.From.PrivateKey)
		if err != nil {
			log.Error("PrivateKey invalid", "error", err)
			return nil, fmt.Errorf("PrivateKey invalid:%s", err.Error())
		}
	}

	if crypto.PubkeyToAddress(txRequestInfo.PrivateKey.PublicKey) != txRequestInfo.From {
		return nil, fmt.Errorf("private key does not match address")
	}

	hash, err := api.s.transaction.BlockTransactionSendToSDAG(&txRequestInfo)
	if err != nil {
		return nil, fmt.Errorf("transaction failed:" + err.Error())
	}

	return struct {
		Hash common.Hash
	}{hash}, nil
}

type TransactionRawParam struct {
	RlpData []byte `json:"rlpdata"`
}

//Rlp transaction
func (api *PublicSdagAPI) TransactionRaw(rlpData *TransactionRawParam) (interface{}, error) {
	log.Debug("receive rlp:", rlpData)
	hash, err := api.s.transaction.RlpTransactionSendToSDAG(rlpData.RlpData)
	if err != nil {
		return nil, fmt.Errorf("transaction failed:" + err.Error())
	}

	return struct {
		Hash common.Hash
	}{hash}, nil
	return rlpData, nil
}

func (api *PublicSdagAPI) GetActiveNodeList(accept string) string { //dashboard RPC server function
	if accept != "ok" {
		fmt.Printf("accept params error")
	}
	nodeIdMessage, _ := api.s.SdagNodeIDMessage()
	nodeIdMsg, err := json.Marshal(nodeIdMessage)
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(nodeIdMsg)

}

// GeneraterKeyStore genetate keystore
func (api *PublicSdagAPI) GeneraterKeyStore(rpcGenerKeyStore *RpcGenerKeyStore) interface{} {

	log.Debug("RPC GeneraterKeyStore", "receives password", rpcGenerKeyStore.Password)
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		fmt.Println("GenerateKey fail")
		return err.Error()
	}

	// Create the keyfile object with a random UUID.
	id := uuid.NewRandom()
	key := &keystore.Key{
		Id:         id,
		Address:    crypto.PubkeyToAddress(privateKey.PublicKey),
		PrivateKey: privateKey,
	}

	// Encrypt key with passphrase.
	passphrase := rpcGenerKeyStore.Password
	keyjson, err := keystore.EncryptKey(key, passphrase, keystore.StandardScryptN, keystore.StandardScryptP)
	if err != nil {
		fmt.Printf("Error encrypting key: %v\n", err)
		return err.Error()
	}

	//Write to File
	pathtmp := filepath.Join(api.s.sct.ResolvePath("keystore"), "")
	errmakefile := os.MkdirAll(pathtmp, 0777)
	if errmakefile != nil {
		return errmakefile.Error()
	}
	path := filepath.Join(pathtmp, "keystore%d")
	keyFilePath := fmt.Sprintf(path, uint64(time.Now().UnixNano())/1e6)
	if err := ioutil.WriteFile(keyFilePath, keyjson, 0600); err != nil {
		fmt.Printf("Failed to write keyfile to %s: %v\n", keyFilePath, err)
		return err.Error()
	}

	return keyFilePath

}

// GetBalance returns the amount of wei for the given address in the state of the
// given block number. The rpc.LatestBlockNumber and rpc.PendingBlockNumber meta
// block numbers are also allowed.
func (api *PublicSdagAPI) GetBalance(walletAddress *WalletAddress) (interface{}, error) {
	address := common.HexToAddress(walletAddress.WalletAddr)
	State := api.s.blockchain.GetLastState()
	balance := State.GetBalance(address)
	return balance.String(), State.Error()
}

func (api *PublicSdagAPI) GetLocalNodeID(jsonstring string) string {
	if jsonstring != "ok" {
		fmt.Printf("accept params error")
	}
	var localIDAndIP = make([]string, 0)
	nodeIdMessage := api.s.nodeID
	nodeIpMessage, ok := api.s.LocalNodeIP()
	//---------------
	var temp = node.DefaultConfig
	nodePortMessage := temp.P2P.ListenAddr

	if !ok {
		return "Query IP Fail"
	}
	localIDAndIP = append(localIDAndIP, "IP: "+nodeIpMessage+nodePortMessage)
	localIDAndIP = append(localIDAndIP, "ID: "+nodeIdMessage)
	nodeIdMsg, err := json.Marshal(localIDAndIP)
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(nodeIdMsg)
}

func (api *PublicSdagAPI) GetConnectNumber() string {
	_, number := api.s.SdagNodeIDMessage()
	num := strconv.Itoa(number)
	return num
}

func (api *PublicSdagAPI) GetMainBlockNumber() string {

	finalMainBlockSlice, err := storage.ReadTailMainBlockInfo(api.s.chainDb)
	if err != nil {
		return ""
	}
	num := strconv.FormatUint(finalMainBlockSlice.Number, 10)
	return num
}

func (api *PublicSdagAPI) GetSyncStatus() string {
	status := api.s.Status().Status
	switch status {
	case STAT_SYNCING:
		return "syncing"
	case STAT_WORKING:
		return "sync_done"
	case STAT_READY:
		return "sync_ready"
	default:
		return "sync_none"
	}
}
func (api *PublicSdagAPI) GetProgressPercent() string {
	progressPercent := api.s.protocolManager.CalculateProgressPercent()
	return progressPercent
}

func (api *PublicSdagAPI) StopMiner() string {
	api.s.config.Mining = false
	result := api.s.miner.PostStop()
	return result
}

func (api *PublicSdagAPI) StartMiner(rpcMinerInfo *RpcMinerInfo) string {

	if api.s.Status().Status != STAT_WORKING {
		return fmt.Sprintf(`{"Error":"current status cannot be miner. current status=%d"}`, api.s.Status().Status)
	}
	if rpcMinerInfo.Address == "" {
		return "address is empty"
	}
	if rpcMinerInfo.Password == "" {
		return "password is empty"
	}
	address := common.HexToAddress(rpcMinerInfo.Address)
	privatekey, err := api.s.accountManager.FindPrivateKey(address, rpcMinerInfo.Password)
	if err != nil {
		log.Error("get private key error", err)
		return err.Error()
	}
	api.s.config.Mining = true
	api.s.miner.Start(rpcMinerInfo.Address, privatekey)
	return "start ok"
}

func hexString2Address(in string, out *common.Address) error {
	if len(in) >= 2 && in[0] == '0' && (in[1] == 'x' || in[1] == 'X') {
		in = in[2:]
	}
	bytes, err := hex.DecodeString(in)
	if err != nil {
		log.Error("hexString2Address failed: %s", err)
		return err
	}
	if len(bytes) > common.AddressLength {
		log.Error("address too length :", "len", len(bytes))
		return fmt.Errorf("address too length : %d", len(bytes))
	}
	for k, byte1 := range bytes {
		out[k] = byte1
	}
	return nil
}

// PrivateAdminAPI is the collection of Ethereum full node-related APIs
// exposed over the private admin endpoint.
type PrivateAdminAPI struct {
	sdag *Sdag
}

// NewPrivateAdminAPI creates a new API definition for the full node private
// admin methods of the Ethereum service.
func NewPrivateAdminAPI(s *Sdag) *PrivateAdminAPI {
	return &PrivateAdminAPI{sdag: s}
}

func (api *PrivateAdminAPI) Do(data int) int {
	return data
}

func (api *PublicSdagAPI) QueryWallet(jsonstring interface{}) (string, error) {
	if reflect.ValueOf(jsonstring).String() != "ok" {
		fmt.Printf("accept params error")
		return "", nil
	}

	var wallet = make([]string, 0)

	tempWallets := api.s.accountManager.Wallets()

	for _, temp := range tempWallets {
		tempA := temp.Accounts()
		for _, temp2 := range tempA {
			wallet = append(wallet, temp2.Address.String())
		}
	}

	walletsMsg, err := json.Marshal(wallet)
	if err != nil {
		return "", err
	}
	return string(walletsMsg), nil
}

type TimeSlice struct {
	TimeSlice uint64
}

func (api *PublicSdagAPI) GetBlockNumberTimeSlice(timeSlice *TimeSlice) (interface{}, error) {
	hash, err := storage.ReadBlocksHashByTmSlice(api.s.chainDb, timeSlice.TimeSlice)
	if err != nil {
		return nil, fmt.Errorf("ReadBlocksHashByTmSlice err" + err.Error())
	}

	return struct {
		Number int
	}{len(hash)}, nil
}

type Hash struct {
	Hash common.Hash
}

func (api *PublicSdagAPI) GetReceipt(hash Hash) (interface{}, error) {
	Receipt, err := storage.ReadReceiptInfo(api.s.chainDb, hash.Hash)
	if err != nil {
		return nil, fmt.Errorf("GetReceipt error:" + err.Error())
	}
	return Receipt, nil
}

type GetState struct {
	Address common.Address
	Hash    common.Hash
}

func (api *PublicSdagAPI) GetState(getState GetState) (interface{}, error) {
	State := api.s.blockchain.GetLastState()
	hash := State.GetState(getState.Address, getState.Hash)
	return hash, State.Error()
}

type GetCode struct {
	Address common.Address
	Hash    common.Hash
}

func (api *PublicSdagAPI) GetCode(getCode GetCode) (interface{}, error) {
	State := api.s.blockchain.GetLastState()
	code := State.GetCode(getCode.Address)
	return common.Bytes2Hex(code), State.Error()
}

func (api *PublicSdagAPI) GetNonce(getCode GetCode) (interface{}, error) {
	State := api.s.blockchain.GetLastState()
	uint64 := State.GetNonce(getCode.Address)
	return uint64, State.Error()
}

type allBalance struct {
	BalanceMap   map[common.Address]string
	TotalBalance string
	NumberAddr   int
}

func (api *PublicSdagAPI) GetAllBalance() (interface{}, error) {
	State := api.s.blockchain.GetLastState()
	balanceMap := State.GetAllBalance()
	var keys []string
	for k := range balanceMap {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	totalBalance := big.NewInt(0)
	balanceStringMap := make(map[common.Address]string)
	for _, k := range keys {
		addr := common.HexToAddress(k)
		totalBalance.Add(totalBalance, balanceMap[addr])
		balanceStringMap[addr] = balanceMap[addr].String()
	}
	return &allBalance{balanceStringMap, totalBalance.String(), len(balanceStringMap)}, nil
}

// CallArgs represents the arguments for a call.
type CallArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      hexutil.Uint64  `json:"gas"`
	GasPrice hexutil.Big     `json:"gasPrice"`
	Value    hexutil.Big     `json:"value"`
	Data     hexutil.Bytes   `json:"data"`
}

func (api *PublicSdagAPI) Call(ctx context.Context, args CallArgs, blockNumber int64) (hexutil.Bytes, error) {
	result, _, _, err := api.doCall(ctx, args, blockNumber, vm.Config{}, 5*time.Second)
	return (hexutil.Bytes)(result), err
}

func (api *PublicSdagAPI) doCall(ctx context.Context, args CallArgs, blockNumber int64, vmCfg vm.Config, timeout time.Duration) ([]byte, uint64, bool, error) {
	defer func(start time.Time) { log.Debug("Executing EVM call finished", "runtime", time.Since(start)) }(time.Now())

	state, err := api.s.blockchain.GetStateCacheByMainNumber(blockNumber)
	if err != nil {
		return nil, 0, false, fmt.Errorf("doCall error:" + err.Error())
	}

	block, err := api.s.blockchain.GetMainBlockByMainNumber(blockNumber)
	if err != nil {
		return nil, 0, false, fmt.Errorf("doCall error:" + err.Error())
	}

	author, err := block.GetSender()
	if err != nil {
		log.Error("doCall GetSender err:" + err.Error())
		return nil, 0, false, fmt.Errorf("doCall GetSender err:" + err.Error())
	}

	// Set sender address or use a default if none specified
	addr := args.From
	if addr == (common.Address{}) {
		if wallets := api.s.accountManager.Wallets(); len(wallets) > 0 {
			if accounts := wallets[0].Accounts(); len(accounts) > 0 {
				addr = accounts[0].Address
			}
		}
	}
	// Set default gas & gas price if none were set
	gas, gasPrice := uint64(args.Gas), args.GasPrice.ToInt()
	if gas == 0 {
		gas = math.MaxUint64 / 2
	}
	if gasPrice.Sign() == 0 {
		gasPrice = params.DefaultGasPrice
	}

	// Create new call message
	msg := types.NewMessage(addr, args.To, 0, args.Value.ToInt(), gas, gasPrice, args.Data, false)

	// Setup context so it may be cancelled the call has completed
	// or, in case of unmetered gas, setup a context with a timeout.
	var cancel context.CancelFunc
	if timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, timeout)
	} else {
		ctx, cancel = context.WithCancel(ctx)
	}
	// Make sure the context is cancelled when the call has completed
	// this makes sure resources are cleaned up.
	defer cancel()

	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, block, api.s.blockchain, &author)
	// Create a new environment which holds all relevant information
	// about the transaction and calling mechanisms.
	evm := vm.NewEVM(context, state, api.s.blockchain.GetChainConfig(), api.s.blockchain.GetVMConfig())
	// Wait for the context to be done and cancel the evm. Even if the
	// EVM has finished, cancelling may be done (repeatedly)
	go func() {
		<-ctx.Done()
		evm.Cancel()
	}()

	// Setup the gas pool (also for unmetered requests)
	// and apply the message.
	gp := new(core.GasPool).AddGas(math.MaxUint64)
	res, gas, failed, err := core.ApplyMessage(evm, msg, gp)
	return res, gas, failed, err
}
