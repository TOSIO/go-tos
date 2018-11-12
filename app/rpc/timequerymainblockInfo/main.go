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

type BolckTime struct {
	Time string
}

type jsonInfo struct {
	Jsonrpc string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  []BolckTime `json:"params"`
	Id      int         `json:"id"`
}

func main() {

	var blockTime string
	fmt.Println("Please input block hash query block Info!")
	//创始区块Time为："1541038119999",
	fmt.Scanf("%s", &blockTime)

	tempJson := jsonInfo{
		Jsonrpc: "2.0",
		Method:  "sdag_getMainBlockInfo",
		Params: []BolckTime{BolckTime{
			Time: blockTime,
		}},
		Id: 1,
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
	fmt.Printf("%s\n", string(body))
	time.Sleep(20 * time.Second)
}
