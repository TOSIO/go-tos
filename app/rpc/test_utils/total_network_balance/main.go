package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
	"github.com/TOSIO/go-tos/node"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
)

var (
	//8545
	urlString1       = "http://10.10.10.37:8545"
	urlString2       = "http://10.10.20.13:8545"
	urlString3       = "http://10.10.10.32:8545"
	urlString4       = "http://10.10.10.42:8551"
	jsonStringFormat = `
{
"jsonrpc":"2.0",
"method":"sdag_getBalance",
"params":[{"WalletAddr" :"%s"}],
"id":1
}`
	urlString = urlString1
)

type KeyInfo struct {
	Address string
}

type RPCResultStr struct {
	Result string
}

func main() {
	var (
		files []os.FileInfo
		err   error
	)
	filePath := filepath.Join(node.DefaultDataDir(), "keystore")
	files, err = ioutil.ReadDir(filePath)
	if err != nil {
		fmt.Println("ReadDir error:", err)
		return
	}

	keyInfoCh := make(chan KeyInfo, len(files))

	for _, file := range files {
		func(file os.FileInfo) {
			fileName := file.Name()
			fmt.Println("ReadFile :", fileName)
			keyJson, err := ioutil.ReadFile(filepath.Join(filePath, fileName))
			if err != nil {
				fmt.Println(err)
				return
			}
			var keyInfo KeyInfo
			if err := json.Unmarshal(keyJson, &keyInfo); err != nil {
				fmt.Println(err)
				return
			}
			keyInfoCh <- keyInfo
		}(file)
	}

	fmt.Println("Parse all  ReadFile complete")
	total := big.NewInt(0)
	count := 0
	for keyInfo := range keyInfoCh {
		sendString := fmt.Sprintf(jsonStringFormat, keyInfo.Address)
		body, err := httpSend.SendHttp(urlString, sendString)
		if err != nil {
			fmt.Println(err)
			continue
		}
		var result RPCResultStr
		err = json.Unmarshal(body, &result)
		if err != nil {
			fmt.Println("Unmarshal error:", err)
			continue
		}
		balance, ok := new(big.Int).SetString(result.Result, 10)
		if !ok {
			fmt.Println("reply balance Parse error")
			continue
		}
		fmt.Printf("address:%s balance:%s\n", keyInfo.Address, balance.String())
		total.Add(total, balance)
		count++
		if count == len(files) {
			break
		}
	}

	fmt.Printf("total count:%d total balance:%s\n", count, total.String())
	return
}
