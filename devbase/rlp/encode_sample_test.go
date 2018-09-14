package rlp

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
)

/*

type MyType struct {
	Name string
	a, b uint
}

type MyNocoolType struct {
	MyType  MyType
	varTest int
}
*/
type BlockHeader struct {
	Version  uint32
	Time     uint64
	GasLimit uint64
	GasPrice *big.Int
}

type Link struct {
	Amount uint64
}

type Block struct {
	Header BlockHeader
	Links  []Link
}

func TestExampleEncoderTest(test *testing.T) {
	t := Block{Header: BlockHeader{127, 455, 66, nil}}
	t.Links = make([]Link, 3)
	t.Links[0].Amount = 4
	t.Links[1].Amount = 6
	t.Links[2].Amount = 7
	bytes, _ := EncodeToBytes(t)
	fmt.Printf("%v → %X\n", t, bytes)
}

//Encode(w, []uint{xa.a, xa.b}) //只对xa的a、b进行序列化
/*
//EncodeToBytes接口
var t *MyNocoolType // t is nil pointer to MyCoolType
t = &MyCoolType{Name: "foobar", a: 5, b: 6,varTest :100}
bytes, _ = EncodeToBytes(t)
fmt.Printf("%v → %X\n", t, bytes)
*/

type MyCoolTypeAgain struct {
	Name string
	A, B uint
}

/* // EncodeRLP writes x as RLP list [a, b] that omits the Name field.
func (x *MyCoolTypeAgain) EncodeRLP(w io.Writer) (err error) {
	// Note: the receiver can be a nil pointer. This allows you to
	// control the encoding of nil, but it also means that you have to
	// check for a nil receiver.
	if x == nil {
		err = Encode(w, []uint{0, 0})
	} else {
		err = Encode(w, []uint{x.A, x.B})
	}
	return err
}
*/
func TestExampleEncoder(test *testing.T) {

	var t *MyCoolTypeAgain // t is nil pointer to MyCoolType
	bytes, _ := EncodeToBytes(t)
	fmt.Printf("%v → %X\n", t, bytes)

	t = &MyCoolTypeAgain{Name: "foobar", A: 5, B: 6}
	byte, _ := EncodeProxy(t) //Encode
	fmt.Printf("%v → %X\n", t, byte)
	bytesa, err := EncodeToBytes(t)
	if err != nil {
		fmt.Println("error occur")
	}
	fmt.Printf("%v → %X\n", t, bytesa)

	size, r, _ := EncodeToReader(t)
	fmt.Printf("%v → %X,%X\n", t, size, r)
	// Output:
	// <nil> → C0
	// &{foobar 5 6} → C986666F6F6261720506
}
func EncodeProxy(val interface{}) ([]byte, error) {
	b := new(bytes.Buffer)
	err := Encode(b, val)
	return b.Bytes(), err
}
