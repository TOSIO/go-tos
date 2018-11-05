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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/statistics"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
	"github.com/pborman/uuid"
	"io/ioutil"
	"math/big"
	"strconv"
	"os"
	"path/filepath"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/log"

	"github.com/TOSIO/go-tos/sdag/transaction"
	"github.com/TOSIO/go-tos/node"
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
func (api *PublicSdagAPI) Status() string {
	data, err := json.Marshal(api.s.Status())
	if err != nil {
		return ""
	} else {
		return string(data)
	}
}

type accountInfo struct {
	Address    string
	PrivateKey string
	Passphrase string
}

type TransactionInfo struct {
	Form     accountInfo
	To       string
	Amount   string
	GasPrice string
	GasLimit string
}

type MainBlockInfo struct {
	Time uint64
}

type BlockHash struct {
	BlockHash common.Hash
}

type WalletAdress struct {
	Address string
}

type RpcMinerInfo struct {
	Address string
	Password string
}

type RpcGenerKeyStore struct {
	Password string
}

func (api *PublicSdagAPI) GetBlockInfo(jsonString string) string {

	var tempblockInfo BlockHash
	if err := json.Unmarshal([]byte(jsonString), &tempblockInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return err.Error()
	}

	db := api.s.chainDb

	blockInfo := api.s.queryBlockInfo.GetBlockInfo(db, tempblockInfo.BlockHash)

	return blockInfo
}

func (api *PublicSdagAPI) GetMainBlockInfo(jsonString string) string {

	var RPCmainBlockInfo MainBlockInfo
	if err := json.Unmarshal([]byte(jsonString), &RPCmainBlockInfo); err != nil {
		log.Error("JSON unmarshaling failed", "error", err)
		return err.Error()
	}

	tempQueryMainBlockInfo := api.s.queryBlockInfo.GetMainBlockInfo(api.s.chainDb, RPCmainBlockInfo.Time)

	return tempQueryMainBlockInfo
}

func (api *PublicSdagAPI) GetFinalMainBlockInfo(jsonString string) string {

	if jsonString != "ok" {
		fmt.Printf("accept params error")
	}

	mainBlockInfo := api.s.queryBlockInfo.GetFinalMainBlockInfo(api.s.chainDb)

	return mainBlockInfo
}

func (api *PublicSdagAPI) Transaction(jsonString string) string {
	if api.s.Status().Status != STAT_WORKING {
		return fmt.Sprintf(`{"Error":"current status cannot be traded. status=%d","Hash":""}`, api.s.Status().Status)
	}
	result := api.transaction(jsonString)
	byteString, err := json.Marshal(result)
	if err != nil {
		return `{"Error":"result parse error","Hash":""}`
	}
	return string(byteString)
}

type ResultStruct struct {
	Error string
	Hash  common.Hash
}

func (api *PublicSdagAPI) transaction(jsonString string) ResultStruct {
	log.Debug("RPC receives Transaction", "receives jsonString", jsonString)
	var transactionInfo TransactionInfo
	if err := json.Unmarshal([]byte(jsonString), &transactionInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return ResultStruct{Error: err.Error()}
	}
	var txRequestInfo transaction.TransInfo

	txRequestInfo.GasPrice = big.NewInt(params.DefaultGasPrice)
	txRequestInfo.GasLimit = params.DefaultGasLimit

	err := hexString2Address(transactionInfo.Form.Address, &txRequestInfo.From)
	if err != nil {
		return ResultStruct{Error: err.Error()}
	}

	var to common.Address
	err = hexString2Address(transactionInfo.To, &to)
	if err != nil {
		return ResultStruct{Error: err.Error()}
	}
	Amount := new(big.Int)
	var ok bool
	_, ok = Amount.SetString(transactionInfo.Amount, 10)
	if !ok {
		log.Error("Amount is invalid", "Amount", transactionInfo.Amount)
		return ResultStruct{Error: "Amount is invalid"}
	}
	if Amount.Sign() < 0 {
		log.Error("The amount must be positive", "Amount", transactionInfo.Amount)
		return ResultStruct{Error: "The amount must be positive"}
	}
	txRequestInfo.GasPrice, ok = new(big.Int).SetString(transactionInfo.GasPrice, 10)
	if !ok {
		log.Error("GasPrice is invalid", "GasPrice", transactionInfo.GasPrice)
		return ResultStruct{Error: "GasPrice is invalid"}
	}

	txRequestInfo.GasLimit, err = strconv.ParseUint(transactionInfo.GasLimit, 10, 64)
	if err != nil {
		log.Error("GasLimit is invalid", "GasLimit", transactionInfo.GasLimit, "error", err.Error())
		return ResultStruct{Error: "GasLimit is invalid"}
	}

	txRequestInfo.Receiver = append(txRequestInfo.Receiver, transaction.ReceiverInfo{to, Amount})

	if len(transactionInfo.Form.PrivateKey) == 0 {
		if len(transactionInfo.Form.Passphrase) == 0 {
			log.Error("Passphrase/PrivateKey invalid")
			return ResultStruct{Error: "Passphrase/PrivateKey invalid"}
		}
		txRequestInfo.PrivateKey, err = api.s.accountManager.FindPrivateKey(txRequestInfo.From, transactionInfo.Form.Passphrase)
		if err != nil {
			log.Error(err.Error())
			return ResultStruct{Error: err.Error()}
		}
	} else {
		txRequestInfo.PrivateKey, err = crypto.HexToECDSA(transactionInfo.Form.PrivateKey)
		if err != nil {
			log.Error("PrivateKey invalid", "error", err)
			return ResultStruct{Error: "PrivateKey invalid error" + err.Error()}
		}
	}

	if crypto.PubkeyToAddress(txRequestInfo.PrivateKey.PublicKey) != txRequestInfo.From {
		return ResultStruct{Error: "private key does not match address"}
	}

	hash, err := transaction.Transaction(api.s.BlockPool(), api.s.BlockPoolEvent(), &txRequestInfo)
	if err != nil {
		return ResultStruct{Error: err.Error()}
	}
	return ResultStruct{Hash: hash}
}

func (api *PublicSdagAPI) GetActiveNodeList(accept string) string { //dashboard RPC server function
	//emptyC <- struct{}{}
	//statisticsObj.Statistics()
	//<-emptyC
	//return "OK"
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

//keystore 生成
func (api *PublicSdagAPI) GeneraterKeyStore(jsonString string) string {
	//Unmarshal json
	var rpcGenerKeyStore RpcGenerKeyStore
	if err := json.Unmarshal([]byte(jsonString), &rpcGenerKeyStore); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return err.Error()
	}
	if rpcGenerKeyStore.Password==""{
		return "password is empty"
	}
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
	pathtmp := filepath.Join(api.s.sct.ResolvePath("keystore"),"")
	errmakefile := os.MkdirAll(pathtmp, 0777)
	if errmakefile != nil {
		return errmakefile.Error()
	}
	path := filepath.Join(pathtmp,"keystore%d")
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
func (api *PublicSdagAPI) GetBalance(jsonString string) (string, error) {

	//Unmarshal json
	var walletadress WalletAdress
	if err := json.Unmarshal([]byte(jsonString), &walletadress); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return "", err
	}
	address := common.HexToAddress(walletadress.Address)
	//last mainblock info
	tailMainBlockInfo := api.s.blockchain.GetMainTail()
	//find the main timeslice
	sTime := utils.GetMainTime(tailMainBlockInfo.Time)
	//get mainblock info
	mainInfo, err := storage.ReadMainBlock(api.s.chainDb, sTime)
	if err != nil {
		return "", err
	}
	//get  statedb
	state, err := state.New(mainInfo.Root, api.s.stateDb)
	if err != nil {
		return "", err
	}
	bigbalance := state.GetBalance(address)
	balance := bigbalance.String()
	return balance, state.Error()
}

func (api *PublicSdagAPI) GetLocalNodeID(jsonstring string) string {
	if jsonstring != "ok" {
		fmt.Printf("accept params error")
	}
	var localIDAndIP  = make([]string, 0)
	nodeIdMessage := api.s.nodeID
	nodeIpMessage, ok := api.s.LocalNodeIP()
	//---------------
	var temp  = node.DefaultConfig
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

func (api *PublicSdagAPI) GetConnectNumber(jsonstring string) string {
	if jsonstring != "ok" {
		fmt.Printf("accept params error")
	}

	_, number := api.s.SdagNodeIDMessage()
	nodeIdMsg, err := json.Marshal(number)
	if err != nil {
		fmt.Println("error:", err)
	}
	return string(nodeIdMsg)
}

//stop minner
func (api *PublicSdagAPI) StopMiner() string {
	api.s.config.Mining = false
	result := api.s.miner.PostStop()
	return result
}

//start miner
func (api *PublicSdagAPI) StartMiner(jsonString string) string {
	if !api.s.miner.CanMiner(){
		return fmt.Sprintf(`{"Error":"current status cannot be miner. status=%d"}`, api.s.miner.SyncState)
	}
	//Unmarshal json
	var rpcMinerInfo RpcMinerInfo
	if err := json.Unmarshal([]byte(jsonString), &rpcMinerInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return err.Error()
	}
	if rpcMinerInfo.Address==""{
		return "address is empty"
	}
	if rpcMinerInfo.Password==""{
		return "password is empty"
	}
	address :=common.HexToAddress(rpcMinerInfo.Address)
	privatekey,err :=api.s.accountManager.FindPrivateKey(address,rpcMinerInfo.Password)
	if err!=nil{
		log.Error("get private key error", err)
		return err.Error()
	}
	api.s.config.Mining = true
	api.s.miner.Start(rpcMinerInfo.Address,privatekey)
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
