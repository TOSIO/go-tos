package core

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/storage"
)

const (
	genesisDev = `
{
  "Time":1546272000000,
  "InitialAccounts":[{"Address":"0f56913b0468147732b83fc465c4c87d110a4804","Amount":"100000000000000000000000000"},
    {"Address":"32641c78a64ba40691a8ff9757ef3e96c0b26d5e","Amount":"100000000000000000000000000"},
    {"Address":"3ab4bff85b11fffd9e8af7491cb5883d588c5254","Amount":"100000000000000000000000000"},
    {"Address":"4ec0851c54f5fb9fad1f8aaca6422aaf2a6d3742","Amount":"100000000000000000000000000"},
    {"Address":"616670964748c1a7409e6fd13abd342239ce96e6","Amount":"100000000000000000000000000"},
    {"Address":"6ccff7f712d7c7aa6becd8d0ef5c3579e2de5c25","Amount":"100000000000000000000000000"},
    {"Address":"7f80dfe6652dadc77193125387178f9ead6d8a43","Amount":"100000000000000000000000000"},
    {"Address":"9619f6173860a8cc2c2bed4b7de7850681ceb36e","Amount":"100000000000000000000000000"},
    {"Address":"9c3b103dc4b7d1ff2ca756904f6981655013a5c9","Amount":"100000000000000000000000000"},
    {"Address":"a3cd1a391bc4fb5739b098e8723a7f0d453f8ccd","Amount":"100000000000000000000000000"},
{"Address":"d3307c0345c427088f640b2c0242f70c79daa08e","Amount":"100000000000000000000000000"}]
}
`
	genesisTest = `
{
  "Time":1540811581000,
  "InitialAccounts":[{"Address":"0f56913b0468147732b83fc465c4c87d110a4804","Amount":"100000000000000000000000000"},
    {"Address":"32641c78a64ba40691a8ff9757ef3e96c0b26d5e","Amount":"100000000000000000000000000"},
    {"Address":"3ab4bff85b11fffd9e8af7491cb5883d588c5254","Amount":"100000000000000000000000000"},
    {"Address":"4ec0851c54f5fb9fad1f8aaca6422aaf2a6d3742","Amount":"100000000000000000000000000"},
    {"Address":"616670964748c1a7409e6fd13abd342239ce96e6","Amount":"100000000000000000000000000"},
    {"Address":"6ccff7f712d7c7aa6becd8d0ef5c3579e2de5c25","Amount":"100000000000000000000000000"},
    {"Address":"7f80dfe6652dadc77193125387178f9ead6d8a43","Amount":"100000000000000000000000000"},
    {"Address":"9619f6173860a8cc2c2bed4b7de7850681ceb36e","Amount":"100000000000000000000000000"},
    {"Address":"9c3b103dc4b7d1ff2ca756904f6981655013a5c9","Amount":"100000000000000000000000000"},
    {"Address":"a3cd1a391bc4fb5739b098e8723a7f0d453f8ccd","Amount":"100000000000000000000000000"},
{"Address":"d3307c0345c427088f640b2c0242f70c79daa08e","Amount":"100000000000000000000000000"}]
}
`

/*	genesisRinkby = `
{
  "Time":1540811581000,
  "InitialAccounts":[{"Address":"d3307c0345c427088f640b2c0242f70c79daa08e","Amount":"100000000000000000000000000"},
    {"Address":"7f80dfe6652dadc77193125387178f9ead6d8a43","Amount":"100000000000000000000000000"}]
}
`*/
)

func (genesis *Genesis) GetGenesisHash() (common.Hash, error) {
	if genesis.genesisHash == (common.Hash{}) {
		if err := genesis.ReadGenesisConfiguration(); err != nil {
			return genesis.genesisHash, err
		}
		mainBlock, err := storage.ReadMainBlock(genesis.db, 0)
		if err != nil {
			return genesis.genesisHash, err
		}
		genesis.genesisHash = mainBlock.Hash
	}

	return genesis.genesisHash, nil
}
