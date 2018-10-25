package manager

import (
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"encoding/json"
	"github.com/TOSIO/go-tos/sdag/mainchain"
	"math/big"
)

type QueryBlockInfoInterface struct{

}



type MutableInfo struct {
	Status string
	ConfirmItsTimeSlice uint64      //Confirm its time slice
	Difficulty          *big.Int    //self difficulty
	CumulativeDiff      *big.Int    //cumulative difficulty
	MaxLink             uint8
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

func  (q *QueryBlockInfoInterface) GetBlockInfo(h tosdb.Database, hash common.Hash) string {

	//commonHash := common.HexToHash(hash)

	mutableInfo, _ := storage.ReadBlockMutableInfo(h, hash)
	blockStatus := q.GetUserBlockStatus(h, hash)
	Data0 := MutableInfo{
		Status: blockStatus,
		ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
		Difficulty: mutableInfo.Difficulty,
		CumulativeDiff: mutableInfo.CumulativeDiff,
		MaxLink: mutableInfo.MaxLink,
	}

	jsonData, err := json.Marshal(Data0)
	if err != nil {
		return ""
	}

	return string(jsonData)
}

func  (q *QueryBlockInfoInterface) GetMainBlockInfo(h tosdb.Database, slice uint64) string {

	mainInfo, err := storage.ReadMainBlock(h, slice)
	if err != nil {
		return ""
	}

	 mainHash := mainInfo.Hash

	 MainBlockInfo := q.GetBlockInfo(h, mainHash)

	 return MainBlockInfo
}

func  (q *QueryBlockInfoInterface) GetFinalMainBlockInfo (h tosdb.Database) string {

	var mianBlock *mainchain.MainChain
	finalMainBlockSlice := mianBlock.GetMainTail()

	mainInfo, err := storage.ReadMainBlock(h, finalMainBlockSlice.Time)
	if err != nil{
		return ""
	}

	mainHash := mainInfo.Hash

	MainBlockInfo := q.GetBlockInfo(h, mainHash)

	return MainBlockInfo

}

