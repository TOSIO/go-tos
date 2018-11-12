package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
	"time"
)

var urlString = "http://10.10.10.13:8545"

type resultError struct {
	Code    int64
	Message string
	IsError bool
}

type resultInfo struct {
	Jsonrpc string
	Id      uint64
	Error   resultError
	Result  string
}

type jsonInfo struct {
	Jsonrpc string   `json:"jsonrpc"`
	Method  string   `json:"method"`
	Params  []string `json:"params"`
	Id      int      `json:"id"`
}

func main() {

	var blockHash = make([]string, 0)
	blockHash = append(blockHash, "ok")

	tempJson := jsonInfo{
		Jsonrpc: "2.0",
		Method:  "sdag_getFinalMainBlockInfo",
		Params:  blockHash,
		Id:      1,
	}

	c, _ := json.Marshal(tempJson)
	body, err := httpSend.SendHttp(urlString, string(c))
	if err != nil {
		fmt.Println("SendHttp error:", err)
	}

	//fmt.Println("receive: ", string(body))
	var result resultInfo
	err = json.Unmarshal(body, &result)
	//if err != nil {
	//	fmt.Println("Unmarshal error:", err)
	//}
	if result.Error.Code != 0 {
		fmt.Println("result error:", result.Error)
	}
	fmt.Println(string(body))
	time.Sleep(20 * time.Second)
}
