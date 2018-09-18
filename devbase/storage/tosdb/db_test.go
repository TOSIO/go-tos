package tosdb_test

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"

	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
)

func TestMemDB_PutGet(t *testing.T) {
	t.Parallel()
	db := tosdb.NewMemDatabase()
	key := []byte{1, 2, 3}
	value := []byte{10, 20, 30}
	db.Put(key, value)
	fmt.Println(db.Len())
	fmt.Println(db.Keys())
	fmt.Println(db.Has(key))
	fmt.Println(db.Get(key))
	fmt.Println(db.Delete(key))

	fmt.Println(db.Len())
	fmt.Println(db.Get(key))
}

func TestTable_PutGet(t *testing.T) {

	t.Parallel()
	db := tosdb.NewMemDatabase()
	prefix := "Hi"
	table := tosdb.NewTable(db, prefix)
	key := []byte{1, 2, 3}
	value := []byte{10, 20, 30}
	table.Put(key, value)
	fmt.Println(db.Has(key))
	fmt.Println(db.Has(append([]byte(prefix), key...)))

	fmt.Println(db.Get(append([]byte(prefix), key...)))
	fmt.Println(db.Delete(append([]byte(prefix), key...)))

	fmt.Println(db.Get(append([]byte(prefix), key...)))
}

func TestLdb_PutGet(t *testing.T) {

	t.Parallel()
	s := "C:\\Users\\Timothy\\Desktop\\ldbtest"
	db, err := tosdb.NewLDBDatabase(s, 16, 16)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(db.Path())
	key := []byte("key")
	value := []byte("value")
	db.Put([]byte(string(key)+"11"), []byte(string(value)+"11"))
	fmt.Println(db.Has([]byte(string(key) + "?")))

	v, _ := db.Get([]byte(string(key) + "11"))
	fmt.Println(string(v))
	v, _ = db.Get([]byte("%d*"))
	fmt.Println(string(v))
	fmt.Println(db.Delete([]byte(string(key) + "*")))

	fmt.Println(db.Get([]byte(string(key) + "*")))
}

type BlockHeader struct { //注意结构体内需要序列化的数据成员要导出（首字母大写）
	Version  uint32
	Time     uint64
	GasLimit uint64
	GasPrice *big.Int
}

type Link struct {
	Amount uint64
	Hash   common.Hash
}

type Block struct {
	Header BlockHeader
	Links  []Link
	s      string
}

//对复杂的结构体（带有嵌入式数据成员）的序列化
func testExampleEncoderTest() *[]byte {
	bigint := new(big.Int)
	bigint.SetUint64(10000000000000000)
	t := Block{Header: BlockHeader{12788, 1536805295000, 6909096, bigint}}
	for i := 0; i < 1000; i++ {
		t.s += "a"
	}
	t.Links = make([]Link, 3)
	t.Links[0].Amount = 0
	t.Links[1].Amount = 1
	t.Links[2].Amount = 2
	for i, _ := range t.Links {
		for j := 0; j < common.HashLength; j++ {
			t.Links[i].Hash[j] = byte(j)
		}
	}
	bytes, err := rlp.EncodeToBytes(t)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%v → %X\n", t, bytes)
	return &bytes
}

// {{127 455 66 <nil>} [{4} {6} {7}]} → CEC67F8201C74280C6C104C106C107

func TestDecodeExampleTest(test *testing.T) {
	var val Block
	err := rlp.Decode(bytes.NewReader([]byte{0xCE, 0xC6, 0x7F, 0x82, 0x01, 0xC7, 0x42, 0x80, 0xC6, 0xC1, 0x04, 0xC1, 0x06, 0xC1, 0x07}), &val)

	fmt.Printf("with 4 elements: err=%v val=%v\n", err, val)
}

func TestLdbDecode_PutGet(t *testing.T) {

	s := "C:\\Users\\Timothy\\Desktop\\ldbtest"
	db, err := tosdb.NewLDBDatabase(s, 16, 16)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(db.Path())
	key := []byte{0}
	value := testExampleEncoderTest() //value type is *[]byte

	start := time.Now()
	for i := 0; i < 1000; i++ {
		key = []byte{byte(i)}
		db.Put(key, *value)
	}
	end := time.Now()
	delta := end.Sub(start)
	fmt.Println(delta)

	for i := 0; i < 1000; i++ {
		key = []byte{byte(i)}
		v, _ := db.Get(key)
		fmt.Println(v)
	}

	for i := 0; i < 1000; i++ {
		key = []byte{byte(i)}
		db.Delete(key)
	}

	for i := 0; i < 1000; i++ {
		key = []byte{byte(i)}
		v, _ := db.Get(key)
		fmt.Println(v)
	}
}
