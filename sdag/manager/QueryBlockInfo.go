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

type MutableInfo struct {
	Status              string
	ConfirmItsTimeSlice uint64   //Confirm its time slice
	Difficulty          *big.Int //self difficulty
	CumulativeDiff      *big.Int //cumulative difficulty
	MaxLinkHash         common.Hash
}

func (q *QueryBlockInfoInterface) GetUserBlockStatus(h tosdb.Database, hash common.Hash) string {

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)

	if err != nil {
		log.Error("Read block stauts fail")
		return ""
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
		return ""
	}
	blockStatus := q.GetUserBlockStatus(h, hash)
	Data0 := MutableInfo{
		Status:              blockStatus,
		ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
		Difficulty:          mutableInfo.Difficulty,
		CumulativeDiff:      mutableInfo.CumulativeDiff,
		MaxLinkHash:         mutableInfo.MaxLinkHash,
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
		return ""
	}

	mainHash := mainInfo.Hash

	MainBlockInfo := q.GetBlockInfo(h, mainHash)

	return MainBlockInfo

}

//func (q *QueryBlockInfoInterface) GetBlockTxInfo(h tosdb.Database, hash common.Hash) string {
//
//	 var TxBlockInfo  *types.TxBlock
//	 var tempTxBlockInfo []string
//
//	temp := storage.ReadBlock(h, hash)
//	tempTxBlockInfo = append(tempTxBlockInfo, "Header:"+fmt.Sprint(temp.))
//	tempTxBlockInfo = append(tempTxBlockInfo, "AccountNonce:"+fmt.Sprint(TxBlockInfo.AccountNonce))
//	tempTxBlockInfo = append(tempTxBlockInfo, "Links:"+fmt.Sprint(TxBlockInfo.Links))
//	tempTxBlockInfo = append(tempTxBlockInfo, "Outs:"+fmt.Sprint(TxBlockInfo.Outs))
//	tempTxBlockInfo = append(tempTxBlockInfo, "Payload:"+fmt.Sprint(TxBlockInfo.Payload))
//
//	return fmt.Sprintln(tempTxBlockInfo)
//
//}
