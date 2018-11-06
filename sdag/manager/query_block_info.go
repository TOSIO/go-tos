package manager

import (
	"encoding/json"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

type QueryBlockInfoInterface struct {
}

type BlockHeader struct {
	Type     uint     //1 tx, 2 miner
	Time     uint64   //ms  timestamp
	GasPrice *big.Int //tls
	GasLimit uint64   //gas max value
}

//type TxOut struct {
//	Receiver common.Address
//	Amount   *big.Int //tls
//}

type TxBlockInfo struct {
	Status              string
	ConfirmItsTimeSlice uint64   //Confirm its time slice
	Difficulty          *big.Int //self difficulty
	CumulativeDiff      *big.Int //cumulative difficulty
	MaxLinkHash         common.Hash
	Time                uint64
	Links               []common.Hash
	BlockHash           common.Hash
	Amount              *big.Int //tls
	Receiver            common.Address
	Sender              common.Address
	GasPrice            *big.Int
	GasLimit            uint64
}

type PublicBlockInfo struct {
	Status              string
	ConfirmItsTimeSlice uint64   //Confirm its time slice
	Difficulty          *big.Int //self difficulty
	CumulativeDiff      *big.Int //cumulative difficulty
	MaxLinkHash         common.Hash
	Time                uint64
	Links               []common.Hash
	BlockHash           common.Hash
	Sender              common.Address
}

type TailMainBlockInfo struct {
	Hash           common.Hash
	CumulativeDiff *big.Int
	Number         uint64
	Time           uint64
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
	//BlockInfo := storage.ReadBlock(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
		return "Query failure"
	}

	Block := storage.ReadBlock(h, hash)

	BlockSend, err := Block.GetSender()
	if err != nil {
		log.Error("Query failure")
	}

	blockStatus := q.GetUserBlockStatus(h, hash)

	if Block.GetType() == types.BlockTypeTx {
		TxBlock, ok := Block.(*types.TxBlock)
		if !ok {
			return ""
		}

		var tempReceiver common.Address
		var tempAmount *big.Int

		for _, temp := range TxBlock.Outs {
			tempReceiver = temp.Receiver
			tempAmount = temp.Amount
		}

		Data0 := TxBlockInfo{
			Status:              blockStatus,
			ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
			Difficulty:          mutableInfo.Difficulty,
			CumulativeDiff:      mutableInfo.CumulativeDiff,
			MaxLinkHash:         mutableInfo.MaxLinkHash,
			Links:               Block.GetLinks(),
			Time:                Block.GetTime(),
			BlockHash:           Block.GetHash(),
			Amount:              tempAmount,
			Receiver:            tempReceiver,
			Sender:              BlockSend,
			GasPrice:            Block.GetGasPrice(),
			GasLimit:            Block.GetGasLimit(),
		}

		jsonData, err := json.Marshal(Data0)
		if err != nil {
			return ""
		}

		return string(jsonData)
	}

	Data0 := PublicBlockInfo{
		Status:              blockStatus,
		ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
		Difficulty:          mutableInfo.Difficulty,
		CumulativeDiff:      mutableInfo.CumulativeDiff,
		MaxLinkHash:         mutableInfo.MaxLinkHash,
		Links:               Block.GetLinks(),
		Time:                Block.GetTime(),
		BlockHash:           Block.GetHash(),
		Sender:              BlockSend,
	}

	jsonData, err := json.Marshal(Data0)
	if err != nil {
		return ""
	}

	return string(jsonData)
}

func (q *QueryBlockInfoInterface) GetBlockAndMainInfo(h tosdb.Database, hash common.Hash, mainBlockInfo *types.MainBlockInfo) string {

	//commonHash := common.HexToHash(hash)

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)
	//BlockInfo := storage.ReadBlock(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
		return "Query is not possible"
	}

	Block := storage.ReadBlock(h, hash)

	BlockSend, err := Block.GetSender()
	if err != nil {
		log.Error("Query Block Sender Fail")
	}

	blockStatus := q.GetUserBlockStatus(h, hash)

	if Block.GetType() == types.BlockTypeTx {
		TxBlock, ok := Block.(*types.TxBlock)
		if !ok {
			return ""
		}

		var tempReceiver common.Address
		var tempAmount *big.Int

		for _, temp := range TxBlock.Outs {
			tempReceiver = temp.Receiver
			tempAmount = temp.Amount
		}

		Data0 := struct {
			TxBlockInfo
			types.MainBlockInfo
		}{
			TxBlockInfo{
				Status:              blockStatus,
				ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
				Difficulty:          mutableInfo.Difficulty,
				CumulativeDiff:      mutableInfo.CumulativeDiff,
				MaxLinkHash:         mutableInfo.MaxLinkHash,
				Links:               Block.GetLinks(),
				Time:                Block.GetTime(),
				BlockHash:           Block.GetHash(),
				Amount:              tempAmount,
				Receiver:            tempReceiver,
				Sender:              BlockSend,
				GasPrice:            Block.GetGasPrice(),
				GasLimit:            Block.GetGasLimit(),
			},
			*mainBlockInfo,
		}

		jsonData, err := json.Marshal(Data0)
		if err != nil {
			return ""
		}

		return string(jsonData)
	}

	Data0 := struct {
		PublicBlockInfo
		types.MainBlockInfo
	}{
		PublicBlockInfo{
			Status:              blockStatus,
			ConfirmItsTimeSlice: mutableInfo.ConfirmItsTimeSlice,
			Difficulty:          mutableInfo.Difficulty,
			CumulativeDiff:      mutableInfo.CumulativeDiff,
			MaxLinkHash:         mutableInfo.MaxLinkHash,
			Links:               Block.GetLinks(),
			Time:                Block.GetTime(),
			BlockHash:           Block.GetHash(),
			Sender:              BlockSend,
		},
		*mainBlockInfo,
	}

	jsonData, err := json.Marshal(Data0)
	if err != nil {
		return ""
	}

	return string(jsonData)
}

func (q *QueryBlockInfoInterface) GetMainBlockInfo(h tosdb.Database, Time uint64) string {

	Slice := utils.GetMainTime(Time)

	mainInfo, err := storage.ReadMainBlock(h, Slice)
	if err != nil {
		return ""
	}
	mainHash := mainInfo.Hash

	MainBlockInfo := q.GetBlockAndMainInfo(h, mainHash, mainInfo)

	return MainBlockInfo
}

func (q *QueryBlockInfoInterface) GetFinalMainBlockInfo(h tosdb.Database) string {

	finalMainBlockSlice, err := storage.ReadTailMainBlockInfo(h)

	if err != nil {
		return "Query failure"
	}

	MainBlickInfo := TailMainBlockInfo{
		Hash:           finalMainBlockSlice.Hash,
		CumulativeDiff: finalMainBlockSlice.CumulativeDiff,
		Number:         finalMainBlockSlice.Number,
		Time:           finalMainBlockSlice.Time,
	}

	jsonData, err := json.Marshal(MainBlickInfo)
	if err != nil {
		return ""
	}

	return string(jsonData)

}
