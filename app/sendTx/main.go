package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"

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
"params":["{\"Form\":{\"Address\" :\"%s\",\"PublicKey\" :\"%s\",\"PrivateKey\"  :\"%s\"},\"To\":\"%s\",\"Amount\":\"%s\"}"],
"id":1
}`
	passphrase = "12345"
	maxRate    = 10000
)

type accountInfo struct {
	Address    string
	PublicKey  string
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

func main() {
	{
		allAccountList := make([]accountInfo, 0, 10000)
		haveBalanceAccountList := make([]accountInfo, 0, 10000)
		haveBalanceAccountMap := map[string]bool{}

		//noBalanceAccountList:=make([]accountInfo,0,10000)

		files, err := ioutil.ReadDir("./keyStore/")
		if err != nil {
			fmt.Println("ReadDir error:", err)
			return
		}

		for _, file := range files {
			fileName := file.Name()
			fmt.Println("ReadFile :", fileName)
			keyjson, err := ioutil.ReadFile("./keyStore/" + fileName)
			if err != nil {
				fmt.Println("ReadFile error:", err)
				continue
			}

			Key, err := keystore.DecryptKey(keyjson, passphrase)
			if err != nil {
				fmt.Println("DecryptKey error", err)
				continue
			}
			allAccountList = append(allAccountList, accountInfo{Address: Key.Address.Hex(),
				PublicKey:  hex.EncodeToString(crypto.FromECDSAPub(&Key.PrivateKey.PublicKey)),
				PrivateKey: hex.EncodeToString(crypto.FromECDSA(Key.PrivateKey)),
				Balance:    big.NewInt(0),
			})
			fmt.Println("Parse ReadFile :", fileName, "complete")
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
				fromAccount.PublicKey,
				fromAccount.PrivateKey,
				toAccount.Address,
				amount.String(),
				"ether")

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
