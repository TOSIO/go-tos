package utils

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"math/big"
)

func CalculateWork(hash common.Hash) *big.Int{
	// 参考6.1 区块难度计算
	return  big.NewInt(1000)
}