package types

import (
	"testing"

	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/rlp"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/pkg/errors"
	"math/big"
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

	b := []byte{248, 156, 211, 1, 134, 1, 103, 161, 170, 223, 76, 132, 5, 245, 225, 0, 133, 1, 0, 0, 0, 0, 225, 160, 219, 213, 214, 16, 204, 33, 19, 200, 191, 2, 209, 252, 136, 10, 16, 200, 68, 179, 65, 174, 168, 120, 117, 22, 35, 252, 246, 80, 16, 105, 43, 54, 77, 224, 223, 148, 8, 216, 75, 190, 13, 13, 170, 187, 36, 21, 214, 65, 157, 249, 6, 214, 205, 250, 221, 108, 137, 52, 135, 147, 142, 4, 153, 132, 0, 0, 48, 27, 160, 81, 233, 146, 225, 103, 37, 169, 49, 78, 145, 94, 154, 223, 209, 154, 189, 154, 141, 253, 186, 195, 22, 222, 138, 53, 39, 246, 96, 142, 66, 188, 111, 160, 116, 146, 5, 86, 70, 202, 233, 140, 20, 108, 90, 139, 195, 53, 68, 21, 199, 196, 174, 236, 178, 204, 73, 95, 193, 117, 16, 53, 157, 99, 192, 75}

	newTx := new(TxBlock)
	if err := rlp.DecodeBytes(b, newTx); err != nil {
		fmt.Println(err)
	}

	fmt.Println("address:", newTx.Outs[0].Receiver.String())
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

func txBlockExample() (*TxBlock, error) {

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
	tx.Outs = []TxOut{
		{addr, big.NewInt(1000)},
	}

	//5. vm code
	tx.Payload = []byte{0x0, 0x3b, 0x0}

	//6. sign
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	tx.Sign(privateKey)

	addr1, _ := tx.GetSender()
	if addr1 != crypto.PubkeyToAddress(privateKey.PublicKey) {
		return nil, errors.New("sign err")
	}

	return tx, nil
}
