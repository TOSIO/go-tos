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

type BlockInfo struct {
	BlockType         string 		 `json:"block_type"`
	ConfirmStatus     string 		 `json:"confirm_status"`
	IsMian			  bool 			 `json:"is_mian"`
	CofirmCount      uint64 		 `json:"cofirm_count"`
	ConfirmItsNumber uint64          `json:"confirm_its_number"`
	ConfirmIndex     uint64          `json:"confirm_Index"`
	Difficulty       *big.Int        `json:"difficulty"`
	CumulativeDiff   *big.Int        `json:"cumulative_diff"`
	MaxLinkHash      common.Hash     `json:"max_link_hash"`
	Time             uint64          `json:"time"`
	Links            []common.Hash   `json:"links"`
	BlockHash        common.Hash     `json:"block_hash"`
	MainBlockHash    common.Hash `json:"main_block_hash"`
	Amount           *big.Int        `json:"amount"`
	ReceiverAddr         common.Address `json:"receiver_addr"`
	SenderAddr           common.Address `json:"sender_addr"`
	MinerAddr        common.Address `json:"miner_addr"`
	GasPrice         *big.Int        `json:"gas_price"`
	GasUsed          uint64          `json:"gasUsed"`
	ActualTxFee      string 		`json:"actual_tx_fee"`
	GasLimit         uint64          `json:"gas_limit"`
}

type PublicBlockInfo struct {
	Status           *types.BlockstatusTmp          `json:"status"`
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

func (q *QueryBlockInfoInterface) GetUserBlockStatus(h tosdb.Database, hash common.Hash) (*types.BlockstatusTmp, error) {

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)

	if err != nil {
		log.Error("Read block stauts fail")
		return nil, err
	}

	tempBlockStatus := mutableInfo.Status

	blockStatus := types.GetBlockStatus(tempBlockStatus)

	return blockStatus, nil
}

func (q *QueryBlockInfoInterface) GetBlockInfo(h tosdb.Database, hash common.Hash) (*BlockInfo, error) {
	var (
	 tempReceiver = common.Address{}
	 tempAmount = big.NewInt(0)
	 err error
	 mainInfo *types.MainBlockInfo
	 confirmcount uint64
	 mainblcokhash =common.Hash{}
	)
	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)
	if err != nil {
		log.Error("Read Txblock Info fail")
		return nil, err
	}

	Block := storage.ReadBlock(h, hash)

	BlockSend, err := Block.GetSender()
	if err != nil {
		log.Error("Query failure")
		return nil, err
	}

	blockStatus, err := q.GetUserBlockStatus(h, hash)
	if err != nil {
		return nil, err
	}
	//根据区块类型做不同处理
	blockType := Block.GetType()
	switch blockType {
	case types.BlockTypeTx://交易区块
		blockStatus.BType = "txblock"
		TxBlock, ok := Block.(*types.TxBlock)
		if ok {

			for _, temp := range TxBlock.Outs {
				tempReceiver = temp.Receiver
				tempAmount = temp.Amount
			}
		}
	case types.BlockTypeMiner://挖矿区块
		blockStatus.BType = "miner_block"
	case types.BlockTypeGenesis://创世区块
		blockStatus.BType = "genesisBlock"
		blockStatus.IsMain = true
		blockStatus.BlockStatus="Accepted"

	}
	//如果是主块
	if blockStatus.IsMain{
		mainInfo, err = storage.ReadMainBlock(h, mutableInfo.ConfirmItsNumber)
		if err != nil {
			return nil, err
		}
		confirmcount = mainInfo.ConfirmCount
		mainblcokhash = mainInfo.Hash
	}
	AllBlockInfo := &BlockInfo{
		BlockType:        getBlockType(Block.GetType()),
		ConfirmStatus:    blockStatus.BlockStatus,
		IsMian:           blockStatus.IsMain,
		CofirmCount:	  confirmcount,
		ConfirmItsNumber: mutableInfo.ConfirmItsNumber,
		ConfirmIndex:     mutableInfo.ConfirmItsIndex,
		Difficulty:       mutableInfo.Difficulty,
		CumulativeDiff:   mutableInfo.CumulativeDiff,
		MaxLinkHash:      mutableInfo.MaxLinkHash,
		Links:            Block.GetLinks(),
		Time:             Block.GetTime(),
		BlockHash:        Block.GetHash(),
		MainBlockHash:    mainblcokhash,
		Amount:           tempAmount,
		ReceiverAddr:     tempReceiver,
		SenderAddr:       BlockSend,
		MinerAddr:        BlockSend,
		GasPrice:         Block.GetGasPrice(),
		GasLimit:         Block.GetGasLimit(),
	}
	return AllBlockInfo, nil
}

func getBlockType(bolockType types.BlockType) string {
	switch bolockType {
	case types.BlockTypeTx:
		return "txblock"
	case types.BlockTypeMiner:
		return "miner_block"
	case types.BlockTypeGenesis:
		return "genesisBlock"

	}
	return ""
}

func (q *QueryBlockInfoInterface) GetBlockAndMainInfo(h tosdb.Database, hash common.Hash, mainBlockInfo *types.MainBlockInfo) (interface{}, error) {

	//commonHash := common.HexToHash(hash)

	mutableInfo, err := storage.ReadBlockMutableInfo(h, hash)
	//BlockInfo := storage.ReadBlock(h, hash)

	if err != nil {
		log.Error("Read Txblock Info fail")
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
			BlockInfo
			types.MainBlockInfo
		}{
			BlockInfo{
				ConfirmStatus:    blockStatus.BlockStatus,
				ConfirmItsNumber: mutableInfo.ConfirmItsNumber,
				ConfirmIndex:     mutableInfo.ConfirmItsIndex,
				Difficulty:       mutableInfo.Difficulty,
				CumulativeDiff:   mutableInfo.CumulativeDiff,
				MaxLinkHash:      mutableInfo.MaxLinkHash,
				Links:            Block.GetLinks(),
				Time:             Block.GetTime(),
				BlockHash:        Block.GetHash(),
				Amount:           tempAmount,
				ReceiverAddr:         tempReceiver,
				SenderAddr:           BlockSend,
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
}
