package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"

	"github.com/TOSIO/go-tos/app/sendTx/httpSend"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
)

var (
	//8545
	urlString        = "http://localhost:8545"
	jsonStringFormat = `
{
"jsonrpc":"2.0",
"method":"sdag_transaction",
"params":["{\"Form\":{\"Address\" :\"%s\",\"PrivateKey\"  :\"%s\"},\"To\":\"%s\",\"Amount\":\"%s\"}"],
"id":1
}`
	passphrase = "12345"
	maxRate    = 10000
)

type accountInfo struct {
	Address    string
	PrivateKey string
	Balance    *big.Int
}
type resultError struct {
	Code    int64
	message string
	IsError bool
}

type resultInfo struct {
	Jsonrpc string
	Id      uint64
	Error   resultError
	Result  string
}

type informations struct {
	keyjson    []byte
	Key        *keystore.Key
	errRead    error
	errDecrypt error
}

func main() {
	{
		allAccountList := make([]accountInfo, 0, 10000)
		haveBalanceAccountList := make([]accountInfo, 0, 10000)
		haveBalanceAccountMap := map[string]bool{}

		//noBalanceAccountList:=make([]accountInfo,0,10000)

		var (
			files []os.FileInfo
			err   error
		)
		files, err = ioutil.ReadDir("./keyStore/")

		if err != nil {
			fmt.Println("ReadDir error:", err)
			return
		}

		ch := make(chan informations, len(files))

		for _, file := range files {

			go func(file os.FileInfo) {
				var info informations
				fileName := file.Name()
				fmt.Println("ReadFile :", fileName)
				info.keyjson, info.errRead = ioutil.ReadFile("./keyStore/" + fileName)

				info.Key, info.errDecrypt = keystore.DecryptKey(info.keyjson, passphrase)

				allAccountList = append(allAccountList, accountInfo{Address: info.Key.Address.Hex(),
					PrivateKey: hex.EncodeToString(crypto.FromECDSA(info.Key.PrivateKey)),
					Balance:    big.NewInt(0),
				})
				fmt.Println("Parse ReadFile :", fileName, "complete")
				ch <- info
			}(file)

			it := <-ch

			if it.errDecrypt != nil {
				fmt.Println("DecryptKey error")
				continue
			}

			if it.errRead != nil {
				fmt.Println("Readfile error")
				continue
			}

		}

		fmt.Println("Parse all  ReadFile complete")

		allAccountList[0].Balance.SetBytes([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
		haveBalanceAccountList = append(haveBalanceAccountList, allAccountList[0])
		haveBalanceAccountMap[allAccountList[0].Address] = true

		for {
			index := rand.Intn(len(haveBalanceAccountList))
			fromAccount := haveBalanceAccountList[index]
			index = rand.Intn(len(allAccountList))
			toAccount := allAccountList[index]
			amountRatio := big.NewInt(int64(rand.Intn(maxRate)))
			tempInt := big.NewInt(0)
			amount := tempInt.Mul(tempInt.Div(fromAccount.Balance, big.NewInt(int64(maxRate))), amountRatio)

			jsonString := fmt.Sprintf(jsonStringFormat,
				fromAccount.Address,
				fromAccount.PrivateKey,
				toAccount.Address,
				amount.String())

			fmt.Println("send: ", jsonString)

			body, err := httpSend.SendHttp(urlString, jsonString)
			if err != nil {
				fmt.Println("SendHttp error:", err)
				continue
			}

			fmt.Println("receive: ", string(body))
			var result resultInfo
			err = json.Unmarshal(body, &result)
			if err != nil {
				fmt.Println("Unmarshal error:", err)
				continue
			}
			if result.Error.Code != 0 {
				fmt.Println("result error:", result.Error)
				continue
			}
			fromAccount.Balance.Sub(fromAccount.Balance, amount)
			toAccount.Balance.Add(toAccount.Balance, amount)
			if v := haveBalanceAccountMap[toAccount.Address]; !v {
				haveBalanceAccountList = append(haveBalanceAccountList, toAccount)
				haveBalanceAccountMap[toAccount.Address] = true
			}
			fmt.Println("--------------------------------------------------------------------------------------------------")

			total := big.NewInt(0)
			for _, v := range haveBalanceAccountList {
				total.Add(total, v.Balance)
			}

			if total.String() != "4722366482869645213696" {
				fmt.Println(total.String(), "errr------------------------")
			}
		}
	}
}
