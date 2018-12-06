package types

import (
	"testing"

	"github.com/TOSIO/go-tos/devbase/crypto"
	"math/big"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/pkg/errors"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"fmt"
)

//测试交易区块
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


func TestTxBlockForJS(t *testing.T) {
	//b := []byte{248, 84, 211, 1, 134, 1, 103, 130, 173, 158, 15, 132, 5, 245, 225, 0, 133, 1, 0, 0, 0, 0, 225, 160, 138, 111, 133, 246, 207, 27, 15, 196, 83, 152, 153, 179, 252, 126, 119, 181, 69, 122, 218, 66, 86, 4, 74, 183, 209, 80, 209, 158, 176, 202, 186, 123, 129, 177, 218, 217, 148, 149, 247, 25, 64, 129, 89, 21, 202, 12, 121, 37, 93, 105, 188, 74, 168, 35, 119, 188, 151, 131, 1, 4, 106, 48}
	//tx := new(TxBlock).data(false)
	//fmt.Println(tx)
	//rlp.DecodeBytes(b, &tx)
	//fmt.Println(tx)

	b := []byte{248,149,211,1,134,1,103,131,105,24,243,132,5,245,225,0,133,1,0,0,0,0,225,160,236,138,118,169,122,89,160,186,123,131,232,106,188,178,198,247,8,211,214,15,160,44,71,41,35,90,215,64,49,125,38,193,15,217,216,148,149,247,25,64,129,89,21,202,12,121,37,93,105,188,74,168,35,119,188,151,130,4,210,48,128,160,85,146,102,168,183,77,58,170,211,75,167,192,184,131,118,239,145,121,185,25,244,21,105,212,211,91,136,142,120,51,70,103,160,42,22,207,17,32,39,170,28,135,72,66,176,141,181,160,181,242,142,64,240,247,244,94,9,166,187,8,184,29,215,159,239}

	newTx := new(TxBlock)
	if err := rlp.DecodeBytes(b, newTx); err != nil {
		fmt.Println(err)
	}

	fmt.Println("v: ", newTx.V)
	fmt.Println("r: ", newTx.R)
	fmt.Println("s: ", newTx.S)

	privateKey, _ := crypto.HexToECDSA("24213f02ad9cf3a30f7a38752be80bd97fc101d61ca79772713ec875ae47060e")
	newTx.Sign(privateKey)

	fmt.Println(newTx)
	fmt.Println("v: ", newTx.V)
	fmt.Println("r: ", newTx.R)
	fmt.Println("s: ", newTx.S)
}


func txBlockExample() (*TxBlock, error){

	//1. set header
	tx := new(TxBlock)
	tx.Header = BlockHeader{
		1,
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
	tx.Payload = []byte{0x0, 0x3b,0x0,
	}

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

