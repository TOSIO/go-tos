package rlp

import (
	"fmt"
	"github.com\TOSIO\go-tos\devbase\rlp"
)

type MyType struct {
	Name string
	a, b uint
}
type MyNocoolType struct {
	MyType  MyType
	varTest int
}

func ExampleEncoder() {

	//var t *MyNocoolType // t is nil pointer to MyCoolType
	t := MyNocoolType{MyType: MyType{"foobar", 5, 6}, varTest: 10}

	bytes, _ := rlp.EncodeToBytes(t)
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
