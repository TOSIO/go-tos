package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/storage"
)

const genesisDefault = `
{
  "Time":1540811581000,
  "InitialAccounts":[{"Address":"d3307c0345c427088f640b2c0242f70c79daa08e","Amount":"100000000000000000000000000"},
    {"Address":"7f80dfe6652dadc77193125387178f9ead6d8a43","Amount":"100000000000000000000000000"}]
}
`

func (genesis *Genesis) GetGenesisHash() (common.Hash, error) {
	if genesis.genesisHash == (common.Hash{}) {
		if err := genesis.ReadGenesisConfiguration(); err != nil {
			return genesis.genesisHash, err
		}
		mainBlock, err := storage.ReadMainBlock(genesis.db, utils.GetMainTime(genesis.Time))
		if err != nil {
			return genesis.genesisHash, err
		}
		genesis.genesisHash = mainBlock.Hash
	}

	return genesis.genesisHash, nil
}
