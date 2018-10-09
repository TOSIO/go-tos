package types

import (
	"math/big"
	"github.com/TOSIO/go-tos/devbase/common"
	"testing"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/pkg/errors"
)

//测试挖矿区块
func TestMinerBlock(t *testing.T) {

	//tx
	mb, err := minerBlockExample()

	if err != nil {
		t.Errorf("mb err: %s", err)
	}

	if err := blockInterfaceExample(mb); err != nil {
		t.Errorf("block err: %s", err)
	}
}

func minerBlockExample() (*MinerBlock, error){

	//1. set header
	mb := new(MinerBlock)
	mb.Header = BlockHeader{
		BlockTypeMiner,
		utils.GetTimeStamp(),
		big.NewInt(10),
		1222,
	}

	//2. links
	b := []byte{
		0xb2, 0x6f, 0x2b, 0x34, 0x2a, 0xab, 0x24, 0xbc, 0xf6, 0x3e,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0xa2, 0x18, 0xc6, 0xa9, 0x27, 0x4d, 0x30, 0xab, 0x9a, 0x15,
		0x10, 0x00,
	}
	var usedH common.Hash
	usedH.SetBytes(b)
	mb.Links = []common.Hash{
		usedH,
	}

	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	//3.miner
	mb.Miner = crypto.PubkeyToAddress(privateKey.PublicKey)

	//4. NONCE
	mb.Nonce = BlockNonce{}

	//5. sign
	mb.Sign(privateKey)


	addr1, _ := mb.GetSender()
	if addr1 != mb.Miner {
		return nil, errors.New("sign err")
	}

	return mb, nil
}

