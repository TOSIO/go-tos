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
"params":[{"Time" :%d}],
"id":1
}
`

	addr1 = `http://10.10.20.13:8545`
	addr2 = `http://10.10.10.37:8545`
	addr3 = `http://10.10.10.34:8545`
	addr  = addr3
)

type ResultTail struct {
	Result struct{ Hash string }
}

type ResultBlock struct {
	Result struct{ Max_link_hash string }
}

func main() {
	body, err := httpSend.SendHttp(addr, sendFormat1)
	if err != nil {
		fmt.Printf("sendString1 error %s", err.Error())
		return
	}
	//fmt.Println(string(body))
	var resultTail ResultTail
	err = json.Unmarshal(body, &resultTail)
	if err != nil {
		fmt.Println(string(body))
		fmt.Println(err)
		return
	}

	Hash := resultTail.Result.Hash
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
		//fmt.Println(string(body))
		var resultBlock ResultBlock
		err = json.Unmarshal(body, &resultBlock)
		if err != nil {
			fmt.Println(err)
			break
		}
		if common.HexToHash(resultBlock.Result.Max_link_hash) == (common.Hash{}) {
			fmt.Printf("MaxLinkHash:%s\n", resultBlock.Result.Max_link_hash)
			fmt.Printf("hash:%v\n", common.HexToHash(Hash))
			fmt.Printf("hash:%v\n", Hash)
			break
		}
		perHash = Hash
		Hash = resultBlock.Result.Max_link_hash
	}
	fmt.Println(count)
	fmt.Println(perHash)
}
