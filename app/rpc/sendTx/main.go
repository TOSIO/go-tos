package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/node"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/app/rpc/httpSend"
	"github.com/TOSIO/go-tos/services/accounts/keystore"
)

var (
	//8545
	testNet          = true
	urlString1       = "http://47.74.255.165:9545"
	urlString2       = "http://10.10.10.37:8545"
	urlString3       = "http://10.10.10.13:8545"
	jsonStringFormat = `
{
"jsonrpc":"2.0",
"method":"sdag_transaction",
"params":[{"From":{"Address" :"%s","PrivateKey"  :"%s"},"GasPrice":"1000000000","GasLimit":4294967296,"To":"%s","Amount":"%s"}],
"id":1
}`
	urlString          = urlString2
	passphrase         = "12345"
	maxRate            = 10000
	totalCount         = big.NewInt(0)
	lastTotalCount     = big.NewInt(0)
	lastTime           int64
	numberMinute       = 0
	lock               sync.Mutex
	totalAmount        = "100000000"
	haveBalanceAddress = common.HexToAddress("0xd3307c0345c427088f640b2c0242f70c79daa08e")
)

type accountInfo struct {
	Address    common.Address
	PrivateKey string
	Balance    *big.Int
}
type resultError struct {
	Code    int64
	Message string
}

type resultInfo struct {
	Jsonrpc string
	Id      uint64
	Error   resultError
	Result  struct{ Hash string }
}

type informations struct {
	keyjson []byte
	Key     *keystore.Key
}

func main() {
	allAccountList := make([]accountInfo, 0, 10000)
	haveBalanceAccountList := make([]accountInfo, 0, 10000)
	haveBalanceAccountMap := map[common.Address]bool{}

	var (
		files    []os.FileInfo
		err      error
		filePath string
	)
	if testNet {
		filePath = filepath.Join(node.DefaultDataDir(), "tos", "keystore")
	} else {
		filePath = filepath.Join(node.DefaultDataDir(), "testNet", "tos", "keystore")
	}
	files, err = ioutil.ReadDir(filePath)

	if err != nil {
		fmt.Println("ReadDir error:", err)
		return
	}

	ch := make(chan informations)

	count := 0
	for _, file := range files {
		go func(file os.FileInfo) {
			defer func() {
				lock.Lock()
				count++
				if count == len(files) {
					close(ch)
				}
				lock.Unlock()
			}()
			var info informations
			fileName := file.Name()
			fmt.Println("ReadFile :", fileName)
			keyJson, err := ioutil.ReadFile(filepath.Join(filePath, fileName))
			if err != nil {
				fmt.Println("ReadFile :", fileName, " error")
			}

			info.Key, err = keystore.DecryptKey(keyJson, passphrase)
			if err != nil {
				fmt.Println("Parse File :", fileName, "DecryptKey error")
				return
			}

			fmt.Println("Parse File :", fileName, "complete")
			ch <- info
		}(file)
	}

	for info := range ch {
		allAccountList = append(allAccountList, accountInfo{Address: info.Key.Address,
			PrivateKey: hex.EncodeToString(crypto.FromECDSA(info.Key.PrivateKey)),
			Balance:    big.NewInt(0),
		})
	}

	fmt.Println("Parse all  ReadFile complete")

	var haveBalance bool
	for index, Account := range allAccountList {
		if Account.Address == haveBalanceAddress {
			allAccountList[index].Balance.SetString(totalAmount, 10)
			haveBalanceAccountList = append(haveBalanceAccountList, allAccountList[index])
			haveBalance = true
			break
		}
	}
	if !haveBalance {
		fmt.Println("no have " + haveBalanceAddress.String() + " key")
		return
	}
	haveBalanceAccountMap[haveBalanceAddress] = true
	lastTime = time.Now().Unix()

	for {
		var (
			fromIndex   int
			toIndex     int
			fromAccount accountInfo
			toAccount   accountInfo
		)
		for {
			fromIndex = rand.Intn(len(haveBalanceAccountList))
			fromAccount = haveBalanceAccountList[fromIndex]
			toIndex = rand.Intn(len(allAccountList))
			toAccount = allAccountList[toIndex]
			if fromAccount != toAccount {
				break
			}
		}

		amountRatio := big.NewInt(int64(rand.Intn(maxRate)))
		tempInt := big.NewInt(0)
		amount := tempInt.Mul(tempInt.Div(fromAccount.Balance, big.NewInt(int64(maxRate))), amountRatio)
		if amount.Sign() == 0 {
			amount = new(big.Int).Set(fromAccount.Balance)
		}

		jsonString := fmt.Sprintf(jsonStringFormat,
			fromAccount.Address.String(),
			fromAccount.PrivateKey,
			toAccount.Address.String(),
			amount.String())

		//fmt.Println("send: ", jsonString)

		body, err := httpSend.SendHttp(urlString, jsonString)
		if err != nil {
			fmt.Println("SendHttp error:", err)
			continue
		}

		//fmt.Println("receive: ", string(body))
		var result resultInfo
		//fmt.Println(string(body))
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
		if fromAccount.Balance.Sign() == 0 {
			haveBalanceAccountList = append(haveBalanceAccountList[:fromIndex], haveBalanceAccountList[fromIndex+1:]...)
			haveBalanceAccountMap[fromAccount.Address] = false
		}
		//fmt.Println("--------------------------------------------------------------------------------------------------")

		total := big.NewInt(0)
		for _, v := range haveBalanceAccountList {
			total.Add(total, v.Balance)
		}

		if total.String() != totalAmount {
			fmt.Println(total.String(), "error------------------------")
		}
		totalCount = totalCount.Add(totalCount, big.NewInt(1))
		nowTime := time.Now().Unix()
		if lastTime+10 < nowTime {
			temp := big.NewInt(0)
			temp = temp.Sub(totalCount, lastTotalCount)
			tempInt := temp.Int64()
			fmt.Println("========================")
			fmt.Println("totalCount=", totalCount)
			fmt.Println("lastTotalCount=", lastTotalCount)
			fmt.Println("10 s totalCount=", tempInt)
			fmt.Println(tempInt/10, "/s")
			lastTotalCount.Set(totalCount)
			lastTime = nowTime
			numberMinute++
			if numberMinute == 10 {
				//return
			}
		}
		//time.Sleep(time.Nanosecond * 1600000)
		//fmt.Println("----------------------------------------------", totalCount.String(), "-----------------------------------------------------")
	}
}
