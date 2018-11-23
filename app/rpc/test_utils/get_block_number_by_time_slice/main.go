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

	addr1 = `http://10.10.20.13:8545`
	addr2 = `http://10.10.10.37:8545`
	addr3 = `http://10.10.10.34:8545`
	addr4 = `http://10.10.10.32:8545`
	addr5 = `http://10.10.10.42:8551`
	addr  = addr2
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
}
