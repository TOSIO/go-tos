package manager

import (
	//"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

type QueryBlockInfoInterface struct {
}

//type TxOut struct {
//	Receiver common.Address
//	Amount   *big.Int //tls
//}

type TxBlockInfo struct {
	BlockType        types.BlockType `json:"block_type"`
	Status           string          `json:"status"`
	ConfirmItsNumber uint64          `json:"confirm_its_number"`
	Difficulty       *big.Int        `json:"difficulty"`
	CumulativeDiff   *big.Int        `json:"cumulative_diff"`
	MaxLinkHash      common.Hash     `json:"max_link_hash"`
	Time             uint64          `json:"time"`
	Links            []common.Hash   `json:"links"`
	BlockHash        common.Hash     `json:"block_hash"`
	Amount           *big.Int        `json:"amount"`
	Receiver         common.Address  `json:"receiver"`
	Sender           common.Address  `json:"sender"`
	GasPrice         *big.Int        `json:"gas_price"`
	GasUsed         uint64         	 `json:"gasUsed"`
	GasLimit         uint64          `json:"gas_limit"`
}

type PublicBlockInfo struct {
	Status           string         `json:"status"`
	ConfirmItsNumber uint64         `json:"confirm_its_number"`
	Difficulty       *big.Int       `json:"difficulty"`
	CumulativeDiff   *big.Int       `json:"cumulative_diff"`
	MaxLinkHash      common.Hash    `json:"max_link_hash"`
	Time             uint64         `json:"time"`
	Links            []common.Hash  `json:"links"`
	BlockHash        common.Hash    `json:"block_hash"`
	Sender           common.Address `json:"sender"`
	GasPrice         *big.Int       `json:"gas_price"`
	GasLimit         uint64         `json:"gas_limit"`
}

type TailMainBlockInfo struct {
	Hash           common.Hash `json:"hash"`
	CumulativeDiff *big.Int    `json:"cumulative_diff"`
	Number         uint64      `json:"number"`
	Time           uint64      `json:"time"`
}

func (q *QueryBlockInfoInterface) GetUserBlockStatus(h tosdb.Database, hash common.Hash) (string, error) {

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)

	if err != nil {
		log.Error("Read block stauts fail")
		//return "Query is not possible"
		return "", err
	}

	tempBlockStatus := mutableInfo.Status

	blockStatus := types.GetBlockStatus(tempBlockStatus)

	return blockStatus, nil
}

func (q *QueryBlockInfoInterface) GetBlockInfo(h tosdb.Database, hash common.Hash) (*TxBlockInfo, error) {

	//commonHash := common.HexToHash(hash)

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)
	//BlockInfo := storage.ReadBlock(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
		//return "Query failure"
		return "", err
	}

	Block := storage.ReadBlock(h, hash)

	BlockSend, err := Block.GetSender()
	if err != nil {
		log.Error("Query failure")
		return "", err
	}

	blockStatus, err := q.GetUserBlockStatus(h, hash)
	if err != nil {
		return "", err
	}
	var tempReceiver = common.Address{}
	var tempAmount = big.NewInt(0)
	TxBlock, ok := Block.(*types.TxBlock)
	if ok {

		for _, temp := range TxBlock.Outs {
			tempReceiver = temp.Receiver
			tempAmount = temp.Amount
		}
	}

	Data0 := &TxBlockInfo{
		BlockType:        Block.GetType(),
		Status:           blockStatus,
		ConfirmItsNumber: mutableInfo.ConfirmItsNumber,
		Difficulty:       mutableInfo.Difficulty,
		CumulativeDiff:   mutableInfo.CumulativeDiff,
		MaxLinkHash:      mutableInfo.MaxLinkHash,
		Links:            Block.GetLinks(),
		Time:             Block.GetTime(),
		BlockHash:        Block.GetHash(),
		Amount:           tempAmount,
		Receiver:         tempReceiver,
		Sender:           BlockSend,
		GasPrice:         Block.GetGasPrice(),
		GasLimit:         Block.GetGasLimit(),
	}
	return Data0, nil
}

func (q *QueryBlockInfoInterface) GetBlockAndMainInfo(h tosdb.Database, hash common.Hash, mainBlockInfo *types.MainBlockInfo) (interface{}, error) {

	//commonHash := common.HexToHash(hash)

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)
	//BlockInfo := storage.ReadBlock(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
		//return "Query is not possible"
		return "", err
	}

	Block := storage.ReadBlock(h, hash)

	BlockSend, err := Block.GetSender()
	if err != nil {
		log.Error("Query Block Sender Fail")
		return "", err
	}

	blockStatus, err := q.GetUserBlockStatus(h, hash)
	if err != nil {
		return "", err
	}

	if Block.GetType() == types.BlockTypeTx {
		TxBlock, ok := Block.(*types.TxBlock)
		if !ok {
			return "", fmt.Errorf("db data type error")
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
				Status:           blockStatus,
				ConfirmItsNumber: mutableInfo.ConfirmItsNumber,
				Difficulty:       mutableInfo.Difficulty,
				CumulativeDiff:   mutableInfo.CumulativeDiff,
				MaxLinkHash:      mutableInfo.MaxLinkHash,
				Links:            Block.GetLinks(),
				Time:             Block.GetTime(),
				BlockHash:        Block.GetHash(),
				Amount:           tempAmount,
				Receiver:         tempReceiver,
				Sender:           BlockSend,
				GasPrice:         Block.GetGasPrice(),
				GasLimit:         Block.GetGasLimit(),
			},
			*mainBlockInfo,
		}
		return Data0, err
	}

	Data0 := struct {
		PublicBlockInfo
		types.MainBlockInfo
	}{
		PublicBlockInfo{
			Status:           blockStatus,
			ConfirmItsNumber: mutableInfo.ConfirmItsNumber,
			Difficulty:       mutableInfo.Difficulty,
			CumulativeDiff:   mutableInfo.CumulativeDiff,
			MaxLinkHash:      mutableInfo.MaxLinkHash,
			Links:            Block.GetLinks(),
			Time:             Block.GetTime(),
			BlockHash:        Block.GetHash(),
			Sender:           BlockSend,
		},
		*mainBlockInfo,
	}
	return Data0, err
}

func (q *QueryBlockInfoInterface) GetMainBlockInfo(h tosdb.Database, number uint64) (interface{}, error) {
	mainInfo, err := storage.ReadMainBlock(h, number)
	if err != nil {
		return "", err
	}
	mainHash := mainInfo.Hash

	MainBlockInfo, err := q.GetBlockAndMainInfo(h, mainHash, mainInfo)
	if err != nil {
		return "", err
	}

	return MainBlockInfo, nil
}

func (q *QueryBlockInfoInterface) GetFinalMainBlockInfo(h tosdb.Database) (interface{}, error) {

	finalMainBlockSlice, err := storage.ReadTailMainBlockInfo(h)

	if err != nil {
		return "", err
	}

	MainBlickInfo := TailMainBlockInfo{
		Hash:           finalMainBlockSlice.Hash,
		CumulativeDiff: finalMainBlockSlice.CumulativeDiff,
		Number:         finalMainBlockSlice.Number,
		Time:           finalMainBlockSlice.Time,
	}
	return MainBlickInfo, err
	//jsonData, err := json.Marshal(MainBlickInfo)
	//if err != nil {
	//	return "", err
	//}
	//
	//return string(jsonData), nil

}
