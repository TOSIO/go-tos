package rlp

import (
	"bytes"
	"fmt"
	"math/big"
	"testing"
)

type BlockHeaderagain struct {
	Version  uint32
	Time     uint64
	GasLimit uint64
	GasPrice *big.Int
}

type Linkagain struct {
	Amount uint64
}

type Blockagain struct {
	Header BlockHeaderagain
	Links  []Linkagain
}

func TestDecodeExampleTest(test *testing.T) {
	var val Blockagain
	err := Decode(bytes.NewReader([]byte{0xCE, 0xC6, 0x7F, 0x82, 0x01, 0xC7, 0x42, 0x80, 0xC6, 0xC1, 0x04, 0xC1, 0x06, 0xC1, 0x07}), &val)

	fmt.Printf("with 4 elements: err=%v val=%v\n", err, val)
}

type MyCoolTypeagain struct {
	Name string
	A, B uint
}

func TestDecodeExample(test *testing.T) {
	var t MyCoolTypeagain
	err := DecodeBytes([]byte{0xC9, 0x86, 0x66, 0x6F, 0x6F, 0x62, 0x61, 0x72, 0x05, 0x06}, &t)
	fmt.Printf("with 4 elements: err=%v val=%v\n", err, t)
}
