package core

import (
	"encoding/json"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
	"io/ioutil"
	"math/big"
)

func NewGenesis(mainChainI mainchain.MainChainI, db tosdb.Database, stateDb state.Database, InitialFilePath string) *Genesis {
	var genesis Genesis
	genesis.mainChainI = mainChainI
	genesis.db = db
	genesis.stateDb = stateDb
	if len(InitialFilePath) == 0 {
		genesis.InitialFilePath = "./Genesis.json"
	} else {
		genesis.InitialFilePath = InitialFilePath
	}
	return &genesis
}

type InitialAccount struct {
	Address string
	Amount  string //tls
}

type InitialGenesisBlockInfo struct {
	Time            uint64
	InitialAccounts []InitialAccount
}

type Genesis struct {
	db              tosdb.Database
	stateDb         state.Database
	mainChainI      mainchain.MainChainI
	InitialFilePath string
	InitialGenesisBlockInfo
}

var GenesisHash common.Hash

func init() {
	GenesisHash.SetBytes([]byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0x10, 0x00,
	})
}

func (genesis *Genesis) ReadGenesisConfiguration() error {
	jsonString, err := ioutil.ReadFile(genesis.InitialFilePath)
	if err != nil {
		return fmt.Errorf("open file fail fileName %s", genesis.InitialFilePath)
	}

	if err := json.Unmarshal([]byte(jsonString), &genesis.InitialGenesisBlockInfo); err != nil {
		return fmt.Errorf("JSON unmarshaling failed: %s", err)
	}

	return nil
}

func (genesis *Genesis) Genesis() {
	tailHash := genesis.mainChainI.GetTail().Hash
	if tailHash != (common.Hash{}) {
		return
	}

	err := genesis.ReadGenesisConfiguration()
	if err != nil {
		log.Error(err.Error())
		return
	}

	minerBlock := new(types.MinerBlock)
	minerBlock.Header = types.BlockHeader{
		types.BlockTypeMiner,
		genesis.Time,
		common.Big0,
		params.DefaultGasLimit,
	}

	info := minerBlock.GetMutableInfo()
	info.Difficulty = minerBlock.GetDiff()
	info.CumulativeDiff = minerBlock.GetCumulativeDiff()
	info.ConfirmItsTimeSlice = utils.GetMainTime(genesis.Time)
	info.Status = types.BlockMain
	storage.WriteBlock(genesis.db, minerBlock)

	state, err := state.New(common.Hash{}, genesis.stateDb)
	if err != nil {
		log.Error("state.New fail [%s]", err.Error())
		return
	}
	for _, initialAccount := range genesis.InitialAccounts {
		amount := new(big.Int)
		if _, ok := amount.SetString(initialAccount.Amount, 10); !ok {
			log.Error("parse amount  err", "Address", initialAccount.Address, "Amount", initialAccount.Amount)
			continue
		}
		state.AddBalance(common.HexToAddress(initialAccount.Address), amount)
	}

	root, err := state.Commit(false)
	if err != nil {
		log.Error(err.Error())
		return
	}
	err = state.Database().TrieDB().Commit(root, true)
	if err != nil {
		log.Error(err.Error())
		return
	}

	var mainBlock types.MainBlockInfo
	mainBlock.Hash = minerBlock.GetHash()
	mainBlock.Root = root
	err = storage.WriteMainBlock(genesis.db, &mainBlock, info.ConfirmItsTimeSlice)
	if err != nil {
		log.Error(err.Error())
	}
}
