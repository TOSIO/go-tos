package main

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
)

var (
	urlString        = "http://10.10.10.23:8545"
)

type request struct {
	Jsonrpc string `json:"jsonrpc"`
	Method string `json:"method"`
	Params []Prams `json:"params"`
	Id int32 `json:"id"`
}

type Prams struct {
	Address string `json:"address"`
}

type ResultError struct {
	Code    int64
	Message string
	IsError bool
}

type ResultInfo struct {
	Jsonrpc string
	Id      uint64
	Error   ResultError
	Result  string
}
func main()  {
	var address string
	fmt.Println("Enter local wallet adress")
	fmt.Scanf("%s", &address)
	requestparams := Prams{
		Address:address,
	}

	jsonparams  := []Prams{requestparams}

	jsonString :=request{
		Jsonrpc:"2.0",
		Method:"sdag_getBalance",
		Params:jsonparams,
		Id:1,
	}
	body, err := json.Marshal(jsonString)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	fmt.Println("params body: ",string(body))
	resposebody, err := httpSend.SendHttp(urlString, string(body))
	if err != nil {
		fmt.Println("SendHttp error:", err)
		return

	}
	var result ResultInfo
	err = json.Unmarshal(resposebody, &result)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
		return
	}
	if result.Error.Code != 0 {
		fmt.Println("result error:", result.Error)
		return
	}

	fmt.Println("result: ",result.Result)

}
