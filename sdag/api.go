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
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"math/big"
	"strings"

	"github.com/TOSIO/go-tos/devbase/statistics"
	"github.com/TOSIO/go-tos/params"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/log"
	//"github.com/TOSIO/go-tos/sdag/core/storage"
	//"github.com/TOSIO/go-tos/sdag/manager"
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

type accountInfo struct {
	Address    string
	PublicKey  string
	PrivateKey string
	Balance    string
}

type TransactionInfo struct {
	Form   accountInfo
	To     string
	Amount string
}

type BlockHash struct {
	BlockHash           common.Hash
}

type BlockInfo struct {
	//BlockHash           common.Hash
	Status              string
	ConfirmItsTimeSlice string
	Difficulty          string
	CumulativeDiff      string
	MaxLink             string
}

func (api *PublicSdagAPI) Apigetstatus(jsonString string) string {

	jsonString = strings.Replace(jsonString, `\`, "", -1)
	var tempblockInfo BlockHash
	if err := json.Unmarshal([]byte(jsonString), &tempblockInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return err.Error()
	}

	db := api.s.chainDb
	var BlockInfo BlockInfo
	reandBlockInfo, _ := storage.ReadBlockMutableInfo(db, tempblockInfo.BlockHash)
	//var reandBlockInfo *types.MutableInfo
	blockStatus := api.s.BlockPool().GetUserBlockStatus(tempblockInfo.BlockHash)
	jsonData, _ := json.Marshal(blockStatus)
	BlockInfo.Status = string(jsonData)
	jsonData1, _ := json.Marshal(reandBlockInfo.ConfirmItsTimeSlice)
	BlockInfo.ConfirmItsTimeSlice = string(jsonData1)
	jsonData2, _ := json.Marshal(reandBlockInfo.Difficulty)
	BlockInfo.Difficulty = string(jsonData2)
	jsonData3, _ := json.Marshal(reandBlockInfo.CumulativeDiff)
	BlockInfo.CumulativeDiff = string(jsonData3)
	jsonData4, _ := json.Marshal(reandBlockInfo.MaxLink)
	BlockInfo.MaxLink = string(jsonData4)

	returnblockInfo := fmt.Sprintln("BlockStatus: ", BlockInfo.Status,
		"BlockConfirmItsTimeSlice: ", BlockInfo.ConfirmItsTimeSlice,
		"BlockDifficulty: ", BlockInfo.Difficulty,
		"BlockCumulativeDiff: ", BlockInfo.CumulativeDiff,
		"BlockMaxLink: ", BlockInfo.MaxLink,
	)

	return returnblockInfo
}

func (api *PublicSdagAPI) Transaction(jsonString string) string {
	//emptyC <- struct{}{}
	//statisticsObj.Statistics()
	//<-emptyC
	//return "OK"
	jsonString = strings.Replace(jsonString, `\`, "", -1)
	var transactionInfo TransactionInfo
	if err := json.Unmarshal([]byte(jsonString), &transactionInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
		return err.Error()
	}
	var txRequestInfo transaction.TransInfo

	txRequestInfo.GasPrice = big.NewInt(params.DefaultGasPrice)
	txRequestInfo.GasLimit = params.DefaultGasLimit

	err := hexString2Address(transactionInfo.Form.Address, &txRequestInfo.From)
	if err != nil {
		return err.Error()
	}

	var to common.Address
	err = hexString2Address(transactionInfo.To, &to)
	if err != nil {
		return err.Error()
	}
	Amount := new(big.Int)
	_, ok := Amount.SetString(transactionInfo.Amount, 10)
	if !ok {
		log.Error("Amount is invalid: %s", transactionInfo.Amount)
		return "Amount is invalid"
	}

	txRequestInfo.Receiver = append(txRequestInfo.Receiver, transaction.ReceiverInfo{to, Amount})

	txRequestInfo.PrivateKey, err = crypto.HexToECDSA(transactionInfo.Form.PrivateKey)
	if err != nil {
		log.Error("HexToECDSA failed: %s", err)
		return "HexToECDSA failed"
	}

	//emptyC <- struct{}{}
	err = transaction.Transaction(api.s.BlockPool(), api.s.BlockPoolEvent(), &txRequestInfo)
	if err != nil {
		return err.Error()
	}
	//<-emptyC
	return "OK"
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
