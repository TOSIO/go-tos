// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package miner implements Ethereum block creation and mining.
package miner

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/devbase/utils"
	"math/big"
	"crypto/ecdsa"
	"github.com/TOSIO/go-tos/devbase/log"
	"math/rand"
	"time"
	"github.com/TOSIO/go-tos/devbase/crypto"
)

const (
	TxBlockType     = 1
	MinerBlockType  = 2
	defaultGasPrice = 100
	defaultGasLimit = 1 << 32
)

var (
	ismining        chan bool
)

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mineinfo *MinerInfo
}

type MinerInfo struct {
	From       common.Address
	GasPrice   *big.Int //tls
	GasLimit   uint64   //gas max value
	PrivateKey *ecdsa.PrivateKey
	event      *interface{}   //监听外部事件
}



func New(minerinfo *MinerInfo) *Miner {

	mine := &Miner{
		mineinfo: minerinfo,
	}

	//3.links
	//if len(mineBlock.Links) == 0 {
	//	mineBlock.Links = manager.SelectUnverifiedBlock(mineBlock.Links)
	//}
	mine.Start()
 return mine

}

//开始挖矿
func (m *Miner) Start() {
		ismining <- true
		work(m)
}

//挖矿
func work(m *Miner) {
	//get random nonce
	nonce := m.getNonceSeed()
	//循环计算nonce值
	go func() {
	loop:
		for {
			//create mine header
			mineBlock := new(types.MinerBlock)
			mineBlock.Header = types.BlockHeader{
				Type:MinerBlockType,
				Time:(utils.GetTimeStamp()+1)-60000,
				GasPrice:m.mineinfo.GasPrice,
				GasLimit:m.mineinfo.GasLimit,
			}
			//calc nonce and compare diff
			select {
			case mining, _ := <-ismining:
				if !mining {
					log.Trace("stop mine nonce", "nonce")
					break loop
				}
			}
			nonce++
			//compare time
			if mineBlock.GetTime() > uint64(time.Now().UnixNano()){
				//add block
				mineBlock.Nonce = types.EncodeNonce(nonce)
				mineBlock.Miner = crypto.PubkeyToAddress(m.mineinfo.PrivateKey.PublicKey)
				//send block
				m.sender(mineBlock)
			}
			//compare diff value
			if mineBlock.GetDiff().Cmp(MianchainDiff()) < 0 {

			}
		}
	}()
}

//停止挖矿
func (m *Miner) Stop() {
		ismining <- false
		work(m)
}

//发送挖矿结果
func (m *Miner) sender(mineblock *types.MinerBlock) {
	//m.mineinfo.Sign(m.mintinfo.PrivateKey)
}

//获取随机数种子
func(m *Miner)getNonceSeed() (nonce uint64) {
	return rand.Uint64()
}

//main block diff value
func MianchainDiff() *big.Int {
	return big.NewInt(100)
}

func GetPervTail(){
	
}
