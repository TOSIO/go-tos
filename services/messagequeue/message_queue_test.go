package messagequeue

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"testing"
)

func TestCreateMQ(t *testing.T) {

	mq, err := CreateMQ()
	defer mq.Close()
	if err != nil {
		fmt.Println(err)
	}

	info := types.MQBlockInfo{
		BlockHash:      "0xc53cd39235cd917ca52bb54b369652af19c2538060we99b6e34e2f9qc5fa3333",
		TransferDate:   "1543835639999",
		Amount:         "2000",
		GasLimit:       "0",
		GasPrice:       "20",
		SenderAddr:     "0x00000000000000wrewrwerqwerqewrqew0000002",
		ReceiverAddr:   "0x0000000000000000000000000000000000000000",
		IsMiner:        "1",
		Difficulty:     "8988484320",
		CumulativeDiff: "51932442661015086",
	}

	blockStatus := types.MQBlockStatus{
		BlockHash:      "0xc53cd39235cd917ca52bb54b369652af19c2538060we99b6e34e2f9qc5fa3333",
		BlockHigh:      "231231",
		IsMain:         "0",
		ConfirmStatus:  "reject",
		ConfirmDate:    "1543835639588",
		GasUsed:        "12",
		ConfirmedHash:  "0xc53cd39235cd917ca52bb54b369652af19c2538060de99b6e34e2f9dc5fa63bf",
		ConfirmedHigh:  "231231",
		ConfirmedOrder: "22",
	}

	err = mq.Publish("blockInfo", info)
	if err != nil {
		log.Error(err.Error())
	}
	err = mq.Publish("blockStatus", blockStatus)
	if err != nil {
		log.Error(err.Error())
	}
}
