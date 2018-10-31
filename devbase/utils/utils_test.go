package utils

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common/math"
	"math/big"
	"testing"
	"time"
)

func TestGetTOSTimeStamp(t *testing.T) {
	tm := time.Now().UnixNano() / 1e6

	fmt.Println(tm)
	//1537435886798

	for i := 1; i < 100; i++ {
		tosTime := GetTOSTimeStamp()
		fmt.Println(tosTime)
		time.Sleep(1 * time.Millisecond)
		t := TOSTimeStampToTime(tosTime)
		fmt.Println(t)
	}

}

func TestCalculateWork(t *testing.T) {
	hahs, ok := new(big.Int).SetString("8a9358aac5851eb3d0c6418019c13fbd5278", 16)
	if ok {
		fmt.Println("ok")
	} else {
		fmt.Println("error")
	}
	fmt.Println(hahs.String())
	den := new(big.Int).Sub(math.BigPow(2, 128), big.NewInt(1))
	//分子 (hash_little / 2^160)
	num := new(big.Int).Div(hahs, math.BigPow(2, 160))
	hd := new(big.Int).Div(den, num)
	fmt.Println(hd.String())
}
