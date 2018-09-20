package types

import (
	"testing"

	"github.com/TOSIO/go-tos/devbase/crypto"
	"math/big"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/pkg/errors"
	"github.com/TOSIO/go-tos/devbase/utils"
)

func TestTxBlock(t *testing.T) {

	//tx
	tx, err := txBlockExample()

	if err != nil {
		t.Errorf("tx err: %s", err)
	}

	if err := blockInterfaceExample(tx); err != nil {
		t.Errorf("block err: %s", err)
	}
}

func txBlockExample() (*TxBlock, error){

	//1. set header
	tx := new(TxBlock)
	tx.Header = BlockHeader{
		BlockTypeTx,
		utils.GetTOSTimeStamp(),
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
	tx.Links = []common.Hash{
		usedH,
	}

	//3. accoutnonce
	tx.AccountNonce = 100

	//4. txout
	var addr common.Address
	tx.Outs = []TxOut {
		{addr, big.NewInt(1000)},
	}

	//5. vm code
	tx.Payload = []byte{0x0, 0x3b}

	//6. sign
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	tx.Sign(privateKey)


	addr1, _ := tx.GetSender()
	if addr1 != crypto.PubkeyToAddress(privateKey.PublicKey){
		return nil, errors.New("sign err")
	}

	return tx, nil
}
