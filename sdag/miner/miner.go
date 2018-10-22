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
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/mainchain"
	"github.com/TOSIO/go-tos/sdag/manager"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/event"
)

const (
	TxBlockType     = 1
	MinerBlockType  = 2
	defaultGasPrice = 100
	defaultGasLimit = 1 << 32
)

var (
	ismining =  make(chan bool)
	feed 		*event.Feed
	mineBlockI types.Block

)

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mineinfo *MinerInfo
	netstatus   chan int
	mainchin   mainchain.MainChainI
}

type MinerInfo struct {
	From       common.Address
	GasPrice   *big.Int //tls
	GasLimit   uint64   //gas max value
	PrivateKey *ecdsa.PrivateKey
}

type MainChain struct {
	Links []common.Hash // 上一时间片主块hash
	diff   *big.Int
}




func New(minerinfo *MinerInfo,mc mainchain.MainChainI) *Miner {

	mine := &Miner{
		mineinfo: minerinfo,
		netstatus:make(chan int,2),
		mainchin:mc,
	}
	mine.Start()
 return mine

}

//start miner work
func (m *Miner) Start() {
	work(m)
	ismining <- true
}

//miner work
func work(m *Miner) {
	//listen subscribe event
	sub  := feed.Subscribe(m.netstatus)
	defer sub.Unsubscribe()
	//get random nonce
	nonce := m.getNonceSeed()
	//Cumulative count of cycles
	var count uint64
	go func() {
	loop:
		for {
			//create mine header

			//calc nonce and compare diff
			select {
			//Subscribe  external netstatus
			case ev , ok := <-m.netstatus:
				if !ok{
					return
				}
				switch ev {

				case core.NETWORK_CONNECTED://net ok
					ismining <- true
				case core.NETWORK_CLOSED://net closed
					ismining <- false
				}
			case mining, _ := <-ismining:
				if !mining {
					log.Trace("stop mine nonce", "nonce")
					break loop
				}
			}

			mineBlock := new(types.MinerBlock)
			mineBlock.Header = types.BlockHeader{
				Type:MinerBlockType,
				Time:(utils.GetMainTime(utils.GetTimeStamp())+1)*params.TimePeriod - 1 ,
				GasPrice:m.mineinfo.GasPrice,
				GasLimit:m.mineinfo.GasLimit,
			}

			fh ,fDiff := m.mainchin.GetPervTail()
			mineBlock.Links[0] = fh
			mineBlock.Links = append(mineBlock.Links,manager.SelectUnverifiedBlock(3)...)
			search:
			for {
				nonce++
				count++
				//每循环1024次检测主链是否更新
				if count ==1024{
					if len(mineBlock.Links) == 0 {
						mineBlock.Links = manager.SelectUnverifiedBlock(3)
					}
					count = 0
				}

				h ,diff := m.mainchin.GetPervTail()
				//compare diff value
				if diff.Cmp(fDiff) >0 {
					mineBlock.Links[0] = h
				}

				//compare time
				if mineBlock.Header.Time > utils.GetTimeStamp(){
					//add block
					mineBlock.Nonce = types.EncodeNonce(nonce)
					mineBlock.Miner = crypto.PubkeyToAddress(m.mineinfo.PrivateKey.PublicKey)

					//send block
					m.sender(mineBlock)
					break search
				}
			}

		}
	}()
}

//stop miner work
func (m *Miner) Stop() {
	work(m)
	ismining <- false
}

//send miner result
func (m *Miner) sender(mineblock *types.MinerBlock) {
	mineBlockI = mineblock
	mineBlockI.Sign(m.mineinfo.PrivateKey)
	manager.SyncAddBlock(mineblock)
}

//get random nonce
func(m *Miner)getNonceSeed() (nonce uint64) {
	return rand.Uint64()
}





