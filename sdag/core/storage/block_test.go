package storage

import (
	"testing"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	//"io/ioutil"
	"os"
	"github.com/TOSIO/go-tos/devbase/common"
	"math/big"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"errors"
	"fmt"
	//"time"
)

//Hash:  [127 242 125 253 133 17 225 78 67 44 32 180 203 174 113 51 76 140 182 175 221 159 171 16 194 223 76 253 141 216 34 53]
//rlp:  [248 151 204 1 134 1 102 41 244 160 158 10 130 4 198 225 160 178 111 43 52 42 171 36 188 246 62 162 24 198 169 39 77 48 171 154 21 162 24 198 169 39 77 48 171 154 21 16 0 100 217 216 148 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 130 3 232 130 0 59 248 67 27 160 118 127 201 67 45 149 33 195 38 63 102 117 171 167 235 46 165 205 49 23 105 224 233 37 101 171 215 209 160 151 175 167 160 68 55 58 225 12 148 250 55 53 12 136 82 97 202 48 77 175 151 162 188 166 134 89 244 248 142 166 144 40 141 141 240 196 128 128 128 128]
//Time:  1538302189726
func TestReadBlockRLP(t *testing.T) {
	db, _ := newTestLDB()
	hash := common.Hash{186,196,1,115,64,52,252,229,73,154,101,134,144,178,203,252,202,119,189,55,43,243,112,127,249,130,209,152,182,158,210,101}

	fmt.Println(ReadBlockRlp(db, hash))
}

func TestHasBlock(t *testing.T) {
	db, _ := newTestLDB()
	hash := common.Hash{184,196,1,115,64,52,252,229,73,154,101,134,144,178,203,252,202,119,189,55,43,243,112,127,249,130,209,152,182,158,210,101}

	fmt.Println(HasBlock(db, hash))
}

func TestWriteBlock(t *testing.T) {
	db, _ := newTestLDB()
	block, _:= txBlockExample()
	fmt.Println("Hash: ", block.GetHash())
	fmt.Println("rlp: ", block.GetRlp())
	fmt.Println("Time: ", block.GetTime())

	WriteBlock(db, block)

	fmt.Println("Read: ", ReadBlock(db, block.GetHash()))
}

func TestReaderBlockHashByTmSlice(t *testing.T) {
	db, _ := newTestLDB()

	for i := 0; i < 1000 ; i++  {
		block, _:= txBlockExample()
		WriteBlock(db, block)
		fmt.Println("Hash: ", block.GetHash())
		fmt.Println("rlp: ", block.GetRlp())
		fmt.Println("Time: ", block.GetTime())

		//time.Sleep(1 * time.Second)
	}

	hashes, _:= ReadBlocksHashByTmSlice(db, utils.GetMainTime(utils.GetTimeStamp()))
	fmt.Println(len(hashes))

	blocks, _ := ReadBlocks(db, hashes)

	for i, j := range blocks {
		fmt.Println("Block:", i, "data: ", j)
	}
}

func TestReadBlocks(t *testing.T) {

}

func newTestLDB() (* tosdb.LDBDatabase, func()) {
	dirname :=  os.TempDir() + "tosdb_test"  //ioutil.TempDir(os.TempDir(), "tosdb_test_")
	fmt.Println("Path: ", dirname)
	//if err != nil {
	//	panic("failed to create test file: " + err.Error())
	//}
	db, err := tosdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}

func txBlockExample() (*types.TxBlock, error){

	//1. set header
	tx := new(types.TxBlock)
	tx.Header = types.BlockHeader{
		types.BlockTypeTx,
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
	tx.Outs = []types.TxOut {
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
