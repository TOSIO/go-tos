package main

import (
	"fmt"
	"github.com/TOSIO/go-tos/app/sendTx/httpSend"
	"encoding/json"
)

var (
	//8545
	urlString        = "http://127.0.0.1:8545"
	jsonStringFormat = `
{
"jsonrpc":"2.0",
"method":"sdag_getBalance",
"params":["{\"Form\":{\"Address\" :\"%s\",\"PrivateKey\"  :\"%s\"},\"To\":\"%s\",\"Amount\":\"%s\"}"],
"id":1
}`

)


type resultError struct {
	Code    int64
	message string
	IsError bool
}

type resultInfo struct {
	Jsonrpc string
	Id      uint64
	Error   resultError
	Result  string
}

type request struct {
	address string
}



func main() {

	repPram := request{
		address:"d3307c0345c427088f640b2c0242f70c79daa081",
	}
	jsonString := fmt.Sprintf(jsonStringFormat,
		repPram.address,
	)

		go func() {

			body, err := httpSend.SendHttp(urlString, jsonString)
			if err!=nil{
				fmt.Println("send erro")
			}

			var result resultInfo
			err = json.Unmarshal(body, &result)
			if result.Error.Code != 0 {
				fmt.Println("result error:", result.Error)
			}
			fmt.Println(result.Result)
		}()
		


}
