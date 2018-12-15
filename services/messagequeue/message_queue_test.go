package messagequeue

import (
	"testing"
	"fmt"
	"encoding/json"
	//"math/big"
	"github.com/TOSIO/go-tos/sdag/core/types"
	//"github.com/TOSIO/go-tos/devbase/common"
)



func TestCreateMQ(t *testing.T) {

	mq, err := CreateMQ()
	defer mq.Close()
	if err != nil {
		fmt.Println(err)
	}

	info := types.MQBlockInfo{
		BlockHash:      "0xc53cd39235cd917ca52bb54b369652af19c2538060we99b6e34e2f9qc5fa333f",
		TransferDate:   "1543835639999",
		Amount:         "2000",
		GasLimit:       "0",
		GasPrice:       "20",
		SenderAddr:     "0x00000000000000wrewrwerqwerqewrqew0000002",
		ReceiverAddr:   "0x0000000000000000000000000000000000000000",
		IsMiner:  		false,
		Difficult:      "8988484320",
		CumulativeDiff: "51932442661015086",
	}


	//blockStatus := struct {
	//	blockHash string,
	//
	//	//"blockHash":"0xc53cd39235cd917ca52bb54b369652af19c2538060we99b6e34e2f9qc5fa333f",
	//	//"transferDate":"1543835639999",
	//	//"amount":"0",
	//	//"gasLimit":"2000",
	//	//"gasPrice":"20",
	//	//"senderAddr":"0x00000000000000wrewrwerqwerqewrqew0000002",
	//	//"receiverAddr":"0x0000000000000000000000000000000000000000",
	//	//"isminer":"1",
	//	//"difficulty":"8988484320",
	//	//"cumulativeDiff":"51932442661015086"
	//}{
	//
	//}

	by, err := json.Marshal(info)
	fmt.Println(string(by))

	mq.Publish("test-key", string(by))

}