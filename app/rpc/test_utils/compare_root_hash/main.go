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
"params":[{"Time" :"%d"}],
"id":1
}
`

	addr1 = `http://10.10.10.37:8545`
	addr2 = `http://10.10.20.13:8545`
)

type ResultBlockInfo struct {
	Result struct {
		Time uint64
	}
}

type ResultTail struct {
	Result struct {
		Hash   string
		Time   uint64
		Number uint64
	}
}

type ResultMainInfo struct {
	Result struct {
		Root          string
		Max_link_hash string
	}
}

func getTail(addr string) (*ResultTail, error) {
	body, err := httpSend.SendHttp(addr, sendFormat1)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	fmt.Println(addr + " :" + string(body))
	var resultTail ResultTail
	err = json.Unmarshal(body, &resultTail)
	if err != nil {
		fmt.Println(string(body))
		fmt.Println(err)
		return nil, err
	}
	Time := resultTail.Result.Time
	fmt.Printf("%d\n", Time)
	return &resultTail, nil
}

func GetBlockInfoByHash(addr string, hash string) (*ResultBlockInfo, error) {
	sendString2 := fmt.Sprintf(sendFormat2, hash)
	body, err := httpSend.SendHttp(addr, sendString2)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	var resultBlockInfo ResultBlockInfo
	err = json.Unmarshal(body, &resultBlockInfo)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &resultBlockInfo, nil
}

func GetMainInfoByTime(addr string, time uint64) (*ResultMainInfo, error) {
	sendString2 := fmt.Sprintf(sendFormat3, time)
	body, err := httpSend.SendHttp(addr, sendString2)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return nil, err
	}
	var resultMainInfo ResultMainInfo
	err = json.Unmarshal(body, &resultMainInfo)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &resultMainInfo, err
}

func main() {
	tail1, err := getTail(addr1)
	if err != nil {
		fmt.Println(err)
		return
	}
	hash1 := tail1.Result.Hash
	tail2, err := getTail(addr2)
	if err != nil {
		fmt.Println(err)
		return
	}
	hash2 := tail2.Result.Hash
	var count uint64
	if tail1.Result.Number == tail2.Result.Number {
		count = tail1.Result.Number
	} else {
		fmt.Printf("Number1:%d Number2:%d\n", tail1.Result.Number, tail2.Result.Number)
		return
	}

	for {
		block1, err := GetBlockInfoByHash(addr1, hash1)
		if err != nil {
			fmt.Println(err)
			return
		}
		mainBlock1, err := GetMainInfoByTime(addr1, block1.Result.Time)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash1 = mainBlock1.Result.Max_link_hash

		block2, err := GetBlockInfoByHash(addr2, hash2)
		if err != nil {
			fmt.Println(err)
			return
		}
		mainBlock2, err := GetMainInfoByTime(addr2, block2.Result.Time)
		if err != nil {
			fmt.Println(err)
			return
		}
		hash2 = mainBlock2.Result.Max_link_hash

		count--
		fmt.Printf("count=%d\n", count)
		fmt.Println(mainBlock1.Result.Root, mainBlock2.Result.Root)
		if mainBlock1.Result.Root == mainBlock2.Result.Root {
			return
		}
	}
}
