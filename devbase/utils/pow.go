package utils

import (
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/common/math"
)

func CalculateWork(hash common.Hash) *big.Int {
	// 参考6.1 区块难度计算
	//(2^128-1)/hash[16:28], hash_little长度为96 bits(12 bytes)
	//1.big int hash
	littleHash := new(big.Int).SetBytes(hash[16:28])
	if littleHash.Cmp(big.NewInt(0)) == 0 {
		littleHash.SetUint64(1)
	}

	//2.根据公式计算
	//分母 (2^128-1)
	den := new(big.Int).Sub(math.BigPow(2, 128), big.NewInt(1))
	//end value
	hd := new(big.Int).Div(den, littleHash)

	return hd
}
