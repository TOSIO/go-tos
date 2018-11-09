package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
	"github.com/TOSIO/go-tos/devbase/common"
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

	addr1 = `http://10.10.20.13:8545`
	addr2 = `http://10.10.10.37:8545`
	addr  = addr1
)

type RPCResultStr struct {
	Result string
}
type ResultTail struct {
	Hash string
}
type ResultBlock struct {
	MaxLinkHash string
}

func main() {
	body, err := httpSend.SendHttp(addr, sendFormat1)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return
	}
	//fmt.Println(string(body))
	var RPCResult RPCResultStr
	err = json.Unmarshal(body, &RPCResult)
	if err != nil {
		fmt.Println(string(body))
		fmt.Println(err)
		return
	}

	var result ResultTail
	err = json.Unmarshal([]byte(RPCResult.Result), &result)
	if err != nil {
		fmt.Println(err)
		return
	}
	Hash := result.Hash
	var perHash string
	fmt.Println(Hash)

	count := 0
	for {
		count++
		sendString2 := fmt.Sprintf(sendFormat2, Hash)
		body, err := httpSend.SendHttp(addr, sendString2)
		if err != nil {
			fmt.Printf("sendString1 error %s", err.Error())
			break
		}
		var RPCResult RPCResultStr
		err = json.Unmarshal(body, &RPCResult)
		if err != nil {
			fmt.Println(err)
			break
		}
		var result ResultBlock
		err = json.Unmarshal([]byte(RPCResult.Result), &result)
		if err != nil {
			fmt.Println(err)
			break
		}
		if common.HexToHash(result.MaxLinkHash) == (common.Hash{}) {
			fmt.Printf("MaxLinkHash:%s\n", result.MaxLinkHash)
			fmt.Printf("hash:%v\n", common.HexToHash(Hash))
			fmt.Printf("hash:%v\n", Hash)
			break
		}
		perHash = Hash
		Hash = result.MaxLinkHash
	}
	fmt.Println(count)
	fmt.Println(perHash)
}
