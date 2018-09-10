package main

import (
	"fmt"
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
)

// struct ts {
// 	a []byte
// }
// func (s *ts)Write(b []byte) (n int, err error){}
// func (s *ts)Width() (wid int, ok bool){}
// func (s *ts)Precision() (prec int, ok bool){}
// func (s *ts)Flag(c int) bool{}
func main() {
	//bytes.go
	// fmt.Println(common.ToHex([]byte{255, 0x02, 0xff}))
	// fmt.Println(common.FromHex("10020f"))
	// fmt.Println(common.FromHex("0x10020f"))
	// fmt.Println(common.FromHex("0X10020f"))
	// fmt.Println(common.Bytes2Hex([]byte{255, 0x02, 0xff}))
	// fmt.Println(common.Hex2Bytes("0X10020f"))
	// fmt.Println(common.Hex2Bytes("10020f"))
	// fmt.Println(common.CopyBytes([]byte{255, 0x02, 0xff}))
	// fmt.Println(common.Hex2BytesFixed("10020f", 5))
	// fmt.Println(common.Hex2BytesFixed("10020f", 2))
	// fmt.Println(common.RightPadBytes([]byte{255, 0x02, 0xff}, 5))
	// fmt.Println(common.RightPadBytes([]byte{255, 0x02, 0xff}, 2))
	// fmt.Println(common.LeftPadBytes([]byte{255, 0x02, 255}, 5))
	// fmt.Println(common.LeftPadBytes([]byte{255, 0x02, 0xff}, 2))

	//debug.go
	//common.Report("asdfas")
	//common.PrintDepricationWarning("sdfjaljsgld")

	//format.go
	//duration := common.PrettyDuration(time.Hour + 123456789)
	//fmt.Println(time.Hour + 123456789)
	//fmt.Println(duration.String())

	//path.go
	// fmt.Println(common.MakeName("name", "1.0"))
	// fmt.Println(common.FileExist("c:\\Go\\src\\"))
	// fmt.Println(common.FileExist("c:\\Go\\src\\1"))
	// fmt.Println(common.AbsolutePath("c:\\Go\\src\\", "all.bat"))
	// fmt.Println(common.AbsolutePath("", "c:\\Go\\src\\all.bat"))
	// fmt.Println(common.AbsolutePath("asdfds", ".\\main.go"))
	// fmt.Println(common.AbsolutePath("sdfg", "all.bat"))

	//size.go
	// s := common.StorageSize(9237500.34534)
	// fmt.Println(s.String())
	// fmt.Println(s.TerminalString())
	// s = common.StorageSize(9230.34534)
	// fmt.Println(s.String())
	// fmt.Println(s.TerminalString())
	// s = common.StorageSize(923.34534)
	// fmt.Println(s.String())
	// fmt.Println(s.TerminalString())

	//test_utils.go
	// var val []struct {
	// 	Title    string
	// 	released int64
	// 	color    bool
	// 	Actors   []string
	// }
	// common.LoadJSON(".\\test.json", &val)
	// fmt.Println(val)

	//types.go
	fmt.Println(common.BytesToHash([]byte{11, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0xf0,
		12, 34, 56, 0x12}))
	bi := new(big.Int)
	fmt.Println(common.BigToHash(bi.SetBits([]big.Word{0, 135, 134, 98746545,
		654564, 45564, 134, 98746545})))

	hash := common.BigToHash(bi.SetBits([]big.Word{0, 135, 134, 98746545,
		654564, 45564, 134, 98746545}))

	fmt.Println(hash.Bytes())
	fmt.Println(hash.Big())
	fmt.Println(hash.Hex())
	fmt.Println(hash.TerminalString())
	fmt.Println(hash.String())

	// var s fmt.State
	// c := rune(0x76)
	// s = new(ts)
	// hash.Format(s, c)
	// fmt.Printf(s, c)
}
