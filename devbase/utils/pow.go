package utils

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"math/big"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common/math"
)

func CalculateWork(hash common.Hash) *big.Int {
	// 参考6.1 区块难度计算
	//(2^128-1)/(hash_little / 2^160), hash_little长度为256 bits(32 bytes)
	//1.little hash
	little_hash := new(big.Int).SetBytes(hash[:])

	//2.根据公式计算
	//分母 (2^128-1)
	den := new(big.Int).Sub(math.BigPow(2, 128), big.NewInt(1))
	//分子 (hash_little / 2^160)
	num := new(big.Int).Div(little_hash, math.BigPow(2, 160))
	hd := new(big.Int).Div(den, num)

	return hd
}