package core

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

func NewGenesis(db tosdb.Database, stateDb state.Database, InitialFilePath string) (*Genesis, error) {
	var genesis Genesis
	genesis.db = db
	genesis.stateDb = stateDb
	if len(InitialFilePath) == 0 {
		genesis.InitialFilePath = "./Genesis.json"
	} else {
		genesis.InitialFilePath = InitialFilePath
	}
	err := genesis.ReadGenesisConfiguration()
	if err != nil {
		log.Error(err.Error())
	}

	types.GenesisTime = genesis.InitialGenesisBlockInfo.Time
	return &genesis, err
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
	InitialFilePath string
	InitialGenesisBlockInfo
	genesisHash common.Hash
}

func (genesis *Genesis) ReadGenesisConfiguration() error {
	jsonString, err := ioutil.ReadFile(genesis.InitialFilePath)
	if err != nil {
		jsonString = []byte(genesisDefault)
		//return fmt.Errorf("open file fail fileName %s", genesis.InitialFilePath)
	}

	if err := json.Unmarshal(jsonString, &genesis.InitialGenesisBlockInfo); err != nil {
		return fmt.Errorf("JSON unmarshaling failed: %s", err)
	}

	return nil
}

func (genesis *Genesis) Genesis() (*types.TailMainBlockInfo, error) {
	for {
		nowTime := utils.GetTimeStamp()
		if genesis.Time+params.TimePeriod < nowTime {
			break
		}
		log.Warn("wait genesis", "wait time(ms)", genesis.Time+params.TimePeriod-nowTime)
		time.Sleep(time.Second)
	}

	genesisBlock := new(types.GenesisBlock)
	genesisBlock.Header = types.BlockHeader{
		types.BlockTypeGenesis,
		genesis.Time,
		common.Big0,
		params.DefaultGasLimit,
	}

	state, err := state.New(common.Hash{}, genesis.stateDb)
	if err != nil {
		log.Error("state.New fail [%s]", err.Error())
		return nil, err
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
		return nil, err
	}
	err = state.Database().TrieDB().Commit(root, true)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	genesisBlock.RootHash = root
	info := genesisBlock.GetMutableInfo()
	info.Difficulty = genesisBlock.GetDiff()
	info.CumulativeDiff = genesisBlock.GetDiff()
	info.ConfirmItsNumber = 0
	info.Status = types.BlockMain | types.BlockTmpMaxDiff | types.BlockApply
	genesisBlock.SetMutableInfo(info)

	storage.WriteBlock(genesis.db, genesisBlock)

	var mainBlock types.MainBlockInfo
	mainBlock.Hash = genesisBlock.GetHash()
	mainBlock.Root = root

	err = storage.WriteMainBlock(genesis.db, &mainBlock, 0)
	if err != nil {
		log.Error(err.Error())
		return nil, err
	}

	var tailMainBlockInfo types.TailMainBlockInfo
	tailMainBlockInfo.Hash = genesisBlock.GetHash()
	tailMainBlockInfo.CumulativeDiff = genesisBlock.GetCumulativeDiff()
	tailMainBlockInfo.Time = genesisBlock.GetTime()
	tailMainBlockInfo.Number = 0
	err = storage.WriteTailMainBlockInfo(genesis.db, &tailMainBlockInfo)
	if err != nil {
		log.Error(err.Error())
	}
	err = storage.WriteTailBlockInfo(genesis.db, &tailMainBlockInfo)
	if err != nil {
		log.Error(err.Error())
	}

	genesis.genesisHash = genesisBlock.GetHash()
	return &tailMainBlockInfo, nil
}
