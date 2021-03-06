package blockboard

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/app/rpc/httpSend"
)

var (
	urlString        = "http://localhost:8545"
	GetConnectNumber ="sdag_getConnectNumber"
	GetMainBlockNumber ="sdag_getMainBlockNumber"
	GetSyncStatus = "sdag_getSyncStatus"
	GetProgressPercent ="sdag_getProgressPercent"
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
	Jsonrpc string `json:"jsonrpc"`
	Id      uint64 `json:"id"`
	Error   ResultError `json:"error"`
	Result  string `json:"result"`
}
func GetSdagInfo(method string)  string{

	jsonString :=request{
		Jsonrpc:"2.0",
		Method:method,
		Params:[]Prams{},
		Id:1,
	}
	body, err := json.Marshal(jsonString)
	if err != nil {
		fmt.Println("Error: ", err)
	}
	resposebody, err := httpSend.SendHttp(urlString, string(body))
	if err != nil {
		fmt.Println("SendHttp error:", err)
		return ""

	}
	var result ResultInfo
	err = json.Unmarshal(resposebody, &result)
	if err != nil {
		fmt.Println("Unmarshal error:", err)
		return ""
	}
	if result.Error.Code != 0 {
		fmt.Println("result error:", result.Error)
		return ""
	}

	return result.Result

}
