package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
)

var (
	sendFormat1 = `{
"jsonrpc":"2.0",
"method":"sdag_getFinalMainBlockInfo",
"params":["ok"],
"id":1
}
`
	sendFormat2 = `{
"jsonrpc":"2.0",
"method":"sdag_getBlockInfo",
"params":["{\"BlockHash\" :\"%s\"}"],
"id":1
}
`
	sendFormat3 = `{
"jsonrpc":"2.0",
"method":"sdag_getMainBlockInfo",
"params":["{\"Time\" :%d}"],
"id":1
}
`

	addr1 = `http://10.10.10.37:8545`
	addr2 = `http://10.10.20.13:8545`
)

type RPCResultStr struct {
	Result string
}
type ResultTail struct {
	Hash string
	Time uint64
}
type ResultBlockInfo struct {
	Time uint64
}
type ResultMainInfo struct {
	Root        string
	MaxLinkHash string
}

func getTail(addr string) (*ResultTail, error) {
	body, err := httpSend.SendHttp(addr, sendFormat1)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	fmt.Println(string(body))
	var RPCResult RPCResultStr
	err = json.Unmarshal(body, &RPCResult)
	if err != nil {
		fmt.Println(string(body))
		fmt.Println(err)
		return nil, err
	}

	var result ResultTail
	err = json.Unmarshal([]byte(RPCResult.Result), &result)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	Time := result.Time
	fmt.Printf("%d\n", Time)
	return &result, nil
}

func GetBlockInfoByHash(addr string, hash string) (*ResultBlockInfo, error) {
	sendString2 := fmt.Sprintf(sendFormat2, hash)
	body, err := httpSend.SendHttp(addr, sendString2)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	var RPCResult RPCResultStr
	err = json.Unmarshal(body, &RPCResult)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var result ResultBlockInfo
	err = json.Unmarshal([]byte(RPCResult.Result), &result)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &result, nil
}

func GetMainInfoByTime(addr string, time uint64) (*ResultMainInfo, error) {
	sendString2 := fmt.Sprintf(sendFormat3, time)
	body, err := httpSend.SendHttp(addr, sendString2)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	var RPCResult RPCResultStr
	err = json.Unmarshal(body, &RPCResult)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	var result ResultMainInfo
	err = json.Unmarshal([]byte(RPCResult.Result), &result)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &result, err
}

func main() {
	tail1, err := getTail(addr1)
	if err != nil {
		fmt.Println(err)
		return
	}
	hash1 := tail1.Hash
	tail2, err := getTail(addr2)
	if err != nil {
		fmt.Println(err)
		return
	}
	hash2 := tail2.Hash
	count := 0
	for {
		block1, err := GetBlockInfoByHash(addr1, hash1)
		if err != nil {
			fmt.Println(err)
			return
		}
		mainBlock1, err := GetMainInfoByTime(addr1, block1.Time)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash1 = mainBlock1.MaxLinkHash

		block2, err := GetBlockInfoByHash(addr2, hash2)
		if err != nil {
			fmt.Println(err)
			return
		}
		mainBlock2, err := GetMainInfoByTime(addr2, block2.Time)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash2 = mainBlock2.MaxLinkHash

		fmt.Printf("count=%d\n", count)
		fmt.Println(mainBlock1.Root, mainBlock2.Root)
		if mainBlock1.Root == mainBlock2.Root {
			return
		}
		count++
	}
}
