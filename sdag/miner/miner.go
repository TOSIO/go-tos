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
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto"
	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
)

const (
	TxBlockType     = 1
	MinerBlockType  = 2
	defaultGasPrice = 100
	defaultGasLimit = 1 << 32
)

// Miner creates blocks and searches for proof-of-work values.
type Miner struct {
	mineinfo  *MinerInfo
	netstatus chan int
	mainchin  mainchain.MainChainI
	blockPool core.BlockPoolI
	feed      *event.Feed

	mining       int32
	miningCh     chan bool
	quitCh       chan struct{}
	stopMiningCh chan struct{}
	wg           sync.WaitGroup

	mineBlockI types.Block
	coinbase   common.Address
}

type MinerInfo struct {
	From       common.Address
	GasPrice   *big.Int //tls
	GasLimit   uint64   //gas max value
	PrivateKey *ecdsa.PrivateKey
}

func New(pool core.BlockPoolI, minerinfo *MinerInfo, mc mainchain.MainChainI, feed *event.Feed) *Miner {
	//init start
	minerinfo.GasLimit = 2000
	minerinfo.GasPrice = big.NewInt(20)
	PrivateKey, err := crypto.GenerateKey()
	if err != nil {
		fmt.Errorf("PrivateKey err")
		return nil
	}
	//fmt.Println("miner new ..................................")
	log.Debug("miner", "new", minerinfo.GasLimit)
	minerinfo.PrivateKey = PrivateKey
	//init end
	mine := &Miner{
		blockPool:    pool,
		mineinfo:     minerinfo,
		netstatus:    make(chan int),
		mainchin:     mc,
		feed:         feed,
		mining:       0,
		miningCh:     make(chan bool),
		quitCh:       make(chan struct{}),
		stopMiningCh: make(chan struct{}),
	}

	return mine

}

//listen chanel mod
func (m *Miner) listen() {
	m.wg.Add(1)
	defer m.wg.Done()
	//fmt.Println("miner listen ..................................")
	log.Debug("miner listen")
	//listen subscribe event
	sub := m.feed.Subscribe(m.netstatus)
	defer sub.Unsubscribe()

	schedule := func(miner *Miner, mining bool) {
		ismining := atomic.LoadInt32(&m.mining)
		if mining {
			if ismining == 0 {
				log.Debug("start miner", "ismining", mining)
				go work(miner)
			}
		} else {
			if ismining == 1 {
				miner.stopMiningCh <- struct{}{}
				log.Debug("Post stop event to miner", "ismining", mining)
			}
		}
	}
	for {
		select {
		//Subscribe  external netstatus
		case ev := <-m.netstatus:
			switch ev {
			case core.NETWORK_CONNECTED: //net ok
				schedule(m, true)
			case core.NETWORK_CLOSED: //net closed
				schedule(m, false)
			case core.SDAGSYNC_SYNCING:
				schedule(m, false)
			case core.SDAGSYNC_COMPLETED:
				schedule(m, true)
			}
		case ismining, ok := <-m.miningCh:
			if ok {
				schedule(m, ismining)
			}
		case <-m.quitCh:
			return
		}
	}
}

//start miner work
func (m *Miner) Start(coinbase common.Address,cofigMing bool) {
	//fmt.Println("miner start ..................................")
	log.Debug("miner", "Start", coinbase)
	m.SetTosCoinbase(coinbase)
	go m.listen()
	if cofigMing {
		m.miningCh <- true
	}

}

//miner work
func work(m *Miner) {

	//fmt.Println("miner work ..................................")
	m.wg.Add(1)
	atomic.StoreInt32(&m.mining, 1)

	clean := func() {
		atomic.StoreInt32(&m.mining, 0)
		m.wg.Done()
	}
	defer clean()

	log.Debug("miner work")
	//get random nonce
	nonce := m.getNonceSeed()

	//Cumulative count of cycles
	var count uint64

	//creat heander
	mineBlock := new(types.MinerBlock)
	mineBlock.Header = types.BlockHeader{
		Type:     MinerBlockType,
		Time:     (utils.GetMainTime(utils.GetTimeStamp())+1)*params.TimePeriod - 1,
		GasPrice: m.mineinfo.GasPrice,
		GasLimit: m.mineinfo.GasLimit,
	}
	//first get prevtail hash  and diff
	fhash, fDiff := m.mainchin.GetPervTail()
	//set PervTailhash  to be best diff
	mineBlock.Links = append(mineBlock.Links, fhash)
	//select params.MaxLinksNum-1 unverifiedblock to links
	mineBlock.Links = append(mineBlock.Links, m.blockPool.SelectUnverifiedBlock(params.MaxLinksNum-1)...)
	// search nonce

	for {
		select {
		case <-m.stopMiningCh:
			log.Debug("Stop mining work")
			return
		default:
			nonce++
			count++
			//每循环1024次检测主链是否更新
			if count == 1024 {
				hash, diff := m.mainchin.GetPervTail()
				//compare diff value
				if diff.Cmp(fDiff) > 0 {
					mineBlock.Links[0] = hash
				}
				count = 0
				continue
			}
			log.Debug("miner", "work", count)
			//compare time
			if mineBlock.Header.Time > utils.GetTimeStamp() {
				log.Debug("miner sender start")
				//add block
				mineBlock.Nonce = types.EncodeNonce(nonce)
				//创币地址即挖矿者本身设置的地址
				mineBlock.Miner = crypto.PubkeyToAddress(m.mineinfo.PrivateKey.PublicKey)

				//send block
				m.sender(mineBlock)
				//break
			}

		}
		time.Sleep(time.Second)
	}

}

//stop miner work
func (m *Miner) Stop() {
	//fmt.Println("miner stop ..................................")
	log.Debug("miner stopped")
	close(m.stopMiningCh)
	m.quitCh <- struct{}{}
	m.wg.Wait()
}

func (m *Miner) PostStop() {
	//fmt.Println("miner stop ..................................")
	log.Debug("miner PostStop")
	close(m.stopMiningCh)
	//m.quitCh <- struct{}{}
	//m.wg.Wait()
}

//send miner result
func (m *Miner) sender(mineblock *types.MinerBlock) {
	//fmt.Println("miner sender ..................................",fmt.Sprintln(mineblock))
	log.Debug("miner sender")
	m.mineBlockI = mineblock
	m.mineBlockI.Sign(m.mineinfo.PrivateKey)
	m.blockPool.EnQueue(mineblock)
}

//get random nonce
func (m *Miner) getNonceSeed() (nonce uint64) {
	return rand.Uint64()
}

func (m *Miner) SetTosCoinbase(coinbase common.Address) {
	m.coinbase = coinbase

}
