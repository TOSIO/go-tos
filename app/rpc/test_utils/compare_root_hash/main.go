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
"params":[{"Number" :%d}],
"id":1
}
`

	addr1 = `http://10.10.10.37:8545`
	addr2 = `http://10.10.10.32:8545`
)

type ResultBlockInfo struct {
	Result struct {
		Number uint64
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

func GetMainInfoByNumber(addr string, time uint64) (*ResultMainInfo, error) {
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

	tail2, err := getTail(addr2)
	if err != nil {
		fmt.Println(err)
		return
	}

	var count uint64
	if tail1.Result.Number == tail2.Result.Number {
		count = tail1.Result.Number
	} else {
		fmt.Printf("Number1:%d Number2:%d\n", tail1.Result.Number, tail2.Result.Number)
		return
	}

	for {
		mainBlock1, err := GetMainInfoByNumber(addr1, count)
		if err != nil {
			fmt.Println(err)
			return
		}
		mainBlock2, err := GetMainInfoByNumber(addr2, count)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Printf("count=%d\n", count)
		fmt.Println(mainBlock1.Result.Root, mainBlock2.Result.Root)
		count--
		if mainBlock1.Result.Root == mainBlock2.Result.Root {
			return
		}
	}
}
