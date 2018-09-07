package main

import (
	"fmt"

	"github.com/TOSIO/go-tos/devbase/common"
)

func main() {
	fmt.Println(common.ToHex([]byte{255, 0x02, 0xff}))
	fmt.Println(common.FromHex("10020f"))
	fmt.Println(common.FromHex("0x10020f"))
	fmt.Println(common.FromHex("0X10020f"))
	fmt.Println(common.Bytes2Hex([]byte{255, 0x02, 0xff}))
	fmt.Println(common.Hex2Bytes("0X10020f"))
	fmt.Println(common.Hex2Bytes("10020f"))
	fmt.Println(common.CopyBytes([]byte{255, 0x02, 0xff}))
	fmt.Println(common.Hex2BytesFixed("10020f", 5))
	fmt.Println(common.Hex2BytesFixed("10020f", 2))
	fmt.Println(common.RightPadBytes([]byte{255, 0x02, 0xff}, 5))
	fmt.Println(common.RightPadBytes([]byte{255, 0x02, 0xff}, 2))
	fmt.Println(common.LeftPadBytes([]byte{255, 0x02, 255}, 5))
	fmt.Println(common.LeftPadBytes([]byte{255, 0x02, 0xff}, 2))
}
