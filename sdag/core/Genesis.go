package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
	"math/big"
)

func New(mainChainI mainchain.MainChainI) {
	var genesis Genesis
	genesis.mainChainI = mainChainI
}

type InitialAccount struct {
	address common.Address
	amount  *big.Int //tls
}

type InitialGenesisBlockInfo struct {
	Time           uint64
	initialAccount []InitialAccount
}

type Genesis struct {
	mainChainI mainchain.MainChainI
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

func (genesis *Genesis) ReadGenesisConfiguration() *InitialGenesisBlockInfo {
	return &InitialGenesisBlockInfo{}
}

func (genesis *Genesis) Genesis() {
	tailHash := genesis.mainChainI.GetTail().Hash
	if tailHash != (common.Hash{}) {
		return
	}

	initialInfo := genesis.ReadGenesisConfiguration()

	//1. set header
	minerBlock := new(types.MinerBlock)
	minerBlock.Header = types.BlockHeader{
		types.BlockTypeMiner,
		initialInfo.Time,
		common.Big0,
		params.DefaultGasLimit,
	}

	//minerBlock.Miner
}
