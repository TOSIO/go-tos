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
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/transaction"
	"math/big"
	"strings"
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

func (api *PublicSdagAPI) DoRequest(data int) int {
	return data
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

func (api *PublicSdagAPI) Transaction(jsonString string) string {
	jsonString = strings.Replace(jsonString, `\`, "", -1)
	var transactionInfo TransactionInfo
	if err := json.Unmarshal([]byte(jsonString), &transactionInfo); err != nil {
		log.Error("JSON unmarshaling failed: %s", err)
	}
	var txRequestInfo transaction.TransInfo

	txRequestInfo.GasPrice.SetUint64(0)
	txRequestInfo.GasLimit = 0

	err := hexString2Address(transactionInfo.Form.Address, &txRequestInfo.From)
	if err != nil {
		return err.Error()
	}

	var to common.Address
	err = hexString2Address(transactionInfo.To, &to)
	if err != nil {
		return err.Error()
	}
	var Amount *big.Int
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

	err = transaction.Transaction(&txRequestInfo)
	if err != nil {
		return err.Error()
	}
	return "OK"
}

func hexString2Address(in string, out *common.Address) error {
	bytes, err := hex.DecodeString(in)
	if err != nil {
		log.Error("hexString2Address failed: %s", err)
		return err
	}
	if len(bytes) > common.AddressLength {
		log.Error("Address too length : %s", len(bytes))
		return fmt.Errorf("Address too length : %s", len(bytes))
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
