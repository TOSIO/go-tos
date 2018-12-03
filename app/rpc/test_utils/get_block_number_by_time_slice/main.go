package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
	"github.com/TOSIO/go-tos/devbase/utils"
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
"params":[{"BlockHash" :"%s"}],
"id":1
}
`

	addr1 = `http://10.10.20.13:8545`
	addr2 = `http://10.10.10.37:8545`
	addr3 = `http://10.10.10.34:8545`
	addr4 = `http://10.10.10.32:8545`
	addr5 = `http://10.10.10.42:8551`
	addr  = addr1
)

type ResultTail struct {
	Result struct{ Hash string }
}

type ResultBlock struct {
	Result struct {
		Time  uint64
		Links []string
	}
}

func GetBlockInfoByHash(addr string, hash string) (*ResultBlock, error) {
	sendString2 := fmt.Sprintf(sendFormat2, hash)
	body, err := httpSend.SendHttp(addr, sendString2)
	if err != nil {
		fmt.Printf("sendString1 error %s\n", err.Error())
		return nil, err
	}
	//stringbody := string(body)
	//fmt.Println(stringbody)
	var resultBlock ResultBlock
	err = json.Unmarshal(body, &resultBlock)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	return &resultBlock, nil
}

var statisticsResult = make(map[uint64]*uint64)
var blockSet = make(map[string]bool)

func statisticsBlock(addr string, hash string) {
	if _, ok := blockSet[hash]; ok {
		return
	}
	blockSet[hash] = true

	ResultBlock, err := GetBlockInfoByHash(addr, hash)

	if err != nil {
		fmt.Println("GetBlockInfoByHash err:" + err.Error())
		return
	}
	for _, hash := range ResultBlock.Result.Links {
		if hash == "0x8e34eb77e8b82c3f7cf16face6b4e188f6287a1762561b8833684f3498c2dc5c" {
			fmt.Println("sdfsd")
		}
		statisticsBlock(addr, hash)
	}

	number, ok := statisticsResult[utils.GetMainTime(ResultBlock.Result.Time)]
	if !ok {
		number = new(uint64)
		statisticsResult[utils.GetMainTime(ResultBlock.Result.Time)] = number
	}
	*number++
}

func main() {
	body, err := httpSend.SendHttp(addr, sendFormat1)
	if err != nil {
		fmt.Printf("sendString1 error %s\n", err.Error())
		return
	}
	fmt.Println(string(body))
	var resultTail ResultTail
	err = json.Unmarshal(body, &resultTail)
	if err != nil {
		fmt.Println(string(body))
		fmt.Println(err)
		return
	}
	statisticsBlock(addr, resultTail.Result.Hash)
	fmt.Printf("block number:%d\n", len(blockSet))
	for k, v := range statisticsResult {
		fmt.Printf("%d:%d\n", k, *v)
	}
}
