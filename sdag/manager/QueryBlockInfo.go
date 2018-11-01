package manager

import (
	"encoding/json"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
	"github.com/TOSIO/go-tos/devbase/utils"
)

type QueryBlockInfoInterface struct {
}

type BlockHeader struct {
	Type     uint //1 tx, 2 miner
	Time     uint64    //ms  timestamp
	GasPrice *big.Int  //tls
	GasLimit uint64    //gas max value
}

//type TxOut struct {
//	Receiver common.Address
//	Amount   *big.Int //tls
//}

type MutableInfo struct {
	Status              string
	ConfirmItsTimeSlice uint64   //Confirm its time slice
	Difficulty          *big.Int //self difficulty
	CumulativeDiff      *big.Int //cumulative difficulty
	MaxLinkHash         common.Hash
	Time                uint64
	Links               []common.Hash
	BlockHash           common.Hash
	//Outs                []TxOut
	//Payload             []byte
}

func (q *QueryBlockInfoInterface) GetUserBlockStatus(h tosdb.Database, hash common.Hash) string {

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)

	if err != nil {
		log.Error("Read block stauts fail")
		return "Query is not possible"
	}

	tempBlockStatus := mutableInfo.Status

	blockStatus := types.GetBlockStatus(tempBlockStatus)

	return blockStatus
}

func (q *QueryBlockInfoInterface) GetBlockInfo(h tosdb.Database, hash common.Hash) string {

	//commonHash := common.HexToHash(hash)

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
		return "Query is not possible"
	}

	TXBlockInfo := storage.ReadBlock(h, hash)

	blockStatus := q.GetUserBlockStatus(h, hash)
	Data0 := MutableInfo{
		Status:              blockStatus,
		ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
		Difficulty:          mutableInfo.Difficulty,
		CumulativeDiff:      mutableInfo.CumulativeDiff,
		MaxLinkHash:         mutableInfo.MaxLinkHash,
		Links:               TXBlockInfo.GetLinks(),
		Time:                TXBlockInfo.GetTime(),
		BlockHash:           TXBlockInfo.GetHash(),

	}

	jsonData, err := json.Marshal(Data0)
	if err != nil {
		return ""
	}

	return string(jsonData)
}

func (q *QueryBlockInfoInterface) GetMainBlockInfo(h tosdb.Database, slice uint64) string {


	mainInfo, err := storage.ReadMainBlock(h, slice)
	if err != nil {
		return ""
	}
	mainHash := mainInfo.Hash

	MainBlockInfo := q.GetBlockInfo(h, mainHash)

	return MainBlockInfo
}

func (q *QueryBlockInfoInterface) GetFinalMainBlockInfo(h tosdb.Database) string {



	finalMainBlockSlice, _ := storage.ReadTailMainBlockInfo(h)
	Time := utils.GetMainTime(finalMainBlockSlice.Time)

	mainInfo, err := storage.ReadMainBlock(h, Time)

	if err != nil {
		return "Query is not possible"
	}

	mainHash := mainInfo.Hash

	MainBlockInfo := q.GetBlockInfo(h, mainHash)

	return MainBlockInfo

}


