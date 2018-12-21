package manager

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/services/messagequeue"
	"strconv"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/event"

	"github.com/TOSIO/go-tos/sdag/mainchain"

	"github.com/TOSIO/go-tos/devbase/common/container"
	"github.com/TOSIO/go-tos/devbase/statistics"

	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

type IsolatedBlock struct {
	Links  []common.Hash //Links
	LinkIt []common.Hash //Link it
	Time   uint32
	RLP    []byte
}

type lackBlock struct {
	LinkIt  []common.Hash //Link it
	Time    int64
	MinTime uint64
}

type protocolManagerI interface {
	RelayBlock(blockRLP []byte) error
	GetBlock(hashBlock common.Hash) error
}

type BlockPool struct {
	pm        protocolManagerI
	db        tosdb.Database
	emptyC    chan struct{}
	blockChan chan types.Block
	mq        *messagequeue.MessageQueue

	MaxSyncTime     uint64
	MaxSyncTimeLock sync.RWMutex

	IsolatedBlockMap map[common.Hash]*IsolatedBlock
	lackBlockMap     map[common.Hash]*lackBlock
	rwlock           sync.RWMutex
	addBlockLock     sync.RWMutex

	maxQueueSize int

	syncStatusFeed *event.Feed
	blockEvent     *event.TypeMux

	syncSub event.Subscription

	newAnnounceSub      *event.TypeMuxSubscription
	localNewBlocksSub   *event.TypeMuxSubscription
	networkNewBlocksSub *event.TypeMuxSubscription
	syncResponseSub     *event.TypeMuxSubscription
	isolateResponseSub  *event.TypeMuxSubscription
	queryUnverifySub    *event.TypeMuxSubscription
	syncStatusSub       chan int

	syncStatus        int
	newBlockAddChan   chan types.Block
	unverifiedAddChan chan common.Hash
	unverifiedDelChan chan common.Hash
	unverifiedBlocks  *container.UniqueList
	listLock          sync.RWMutex

	statisticsAddBlock            statistics.Statistics
	statisticsDeleteIsolatedBlock statistics.Statistics

	mainChainI mainchain.MainChainI
}

func New(mainChain mainchain.MainChainI, chainDb tosdb.Database, feed *event.Feed, blockEvent *event.TypeMux, mq *messagequeue.MessageQueue) *BlockPool {
	pool := &BlockPool{
		mainChainI: mainChain,
		db:         chainDb,

		syncStatusFeed: feed,
		emptyC:         make(chan struct{}, 1),
		blockChan:      make(chan types.Block, 8),
		mq:             mq,

		IsolatedBlockMap: make(map[common.Hash]*IsolatedBlock),
		lackBlockMap:     make(map[common.Hash]*lackBlock),

		maxQueueSize: 50000,

		newBlockAddChan:   make(chan types.Block, 50000),
		unverifiedAddChan: make(chan common.Hash, 1000),
		unverifiedDelChan: make(chan common.Hash, 2000),
		syncStatusSub:     make(chan int),

		blockEvent: blockEvent,
	}
	pool.newAnnounceSub = pool.blockEvent.Subscribe(&core.AnnounceEvent{})
	pool.localNewBlocksSub = pool.blockEvent.Subscribe(&core.LocalNewBlocksEvent{})
	pool.networkNewBlocksSub = pool.blockEvent.Subscribe(&core.NetworkNewBlocksEvent{})
	pool.syncResponseSub = pool.blockEvent.Subscribe(&core.SYNCResponseEvent{})
	pool.isolateResponseSub = pool.blockEvent.Subscribe(&core.IsolateResponseEvent{})
	pool.queryUnverifySub = pool.blockEvent.Subscribe(&core.GetUnverifyBlocksEvent{})
	pool.syncSub = pool.syncStatusFeed.Subscribe(pool.syncStatusSub)
	pool.unverifiedBlocks = container.NewUniqueList(pool.maxQueueSize * 3)
	pool.statisticsAddBlock.Init("add block")
	pool.statisticsDeleteIsolatedBlock.Init("delete isolated block")
	go pool.loop()
	return pool
}

func (p *BlockPool) BlockProcessing() {
	localNewBlocks := make(chan types.Block, 50000)
	isolateResponse := make(chan types.Block, 50000)
	networkNewBlocks := make(chan types.Block, 50000)
	syncResponse := make(chan types.Block, 50000)
	go func() {
		for {
			select {
			case ch := <-p.localNewBlocksSub.Chan():
				if ev, ok := ch.Data.(*core.LocalNewBlocksEvent); ok {
					for _, block := range ev.Blocks {
						localNewBlocks <- block
					}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case ch := <-p.isolateResponseSub.Chan():
				if ev, ok := ch.Data.(*core.IsolateResponseEvent); ok {
					for _, block := range ev.Blocks {
						isolateResponse <- block
					}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case ch := <-p.networkNewBlocksSub.Chan():
				if ev, ok := ch.Data.(*core.NetworkNewBlocksEvent); ok {
					for _, block := range ev.Blocks {
						networkNewBlocks <- block
					}
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case ch := <-p.syncResponseSub.Chan():
				if ev, ok := ch.Data.(*core.SYNCResponseEvent); ok {
					for _, block := range ev.Blocks {
						syncResponse <- block
					}
				}
			}
		}
	}()

	FromNetGoroutineCount := 128
	for i := 0; i < FromNetGoroutineCount; i++ {
		go func() {
			for {
				select {
				case block := <-localNewBlocks:
					p.AddBlock(block, true, false)
					continue
				default:
				}
				select {
				case block := <-isolateResponse:
					p.AddBlock(block, false, false)
					continue
				default:
				}
				select {
				case block := <-networkNewBlocks:
					p.AddBlock(block, true, false)
					continue
				default:
				}
				select {
				case block := <-syncResponse:
					p.AddBlock(block, false, true)
					continue
				default:
				}
				time.Sleep(time.Millisecond)
			}
		}()
	}
}

func (p *BlockPool) TimedRequestForIsolatedBlocks() {
	go func() {
		currentTime := time.Now().Unix()
		lastTime := currentTime
		for {
			select {
			case p.syncStatus = <-p.syncStatusSub:
			default:
			}
			currentTime = time.Now().Unix()

			if lastTime+params.TimePeriod/1000 < currentTime {
				var linksLackBlock []common.Hash
				var blockList []types.Block
				p.MaxSyncTimeLock.RLock()
				MaxSyncTime := p.MaxSyncTime
				p.MaxSyncTimeLock.RUnlock()
				p.rwlock.RLock()
				for key, value := range p.lackBlockMap {
					log.Debug("Request ancestor", "hash", key.String(), "lackBlockMap len", len(p.lackBlockMap))
					if storage.HasBlock(p.db, key) {
						block := storage.ReadBlock(p.db, key)
						if block != nil {
							blockList = append(blockList, block)
							continue
						}
					}
					if !(p.syncStatus == core.SDAGSYNC_SYNCING &&
						utils.GetMainTime(value.MinTime) >= utils.GetMainTime(MaxSyncTime)) {
						linksLackBlock = append(linksLackBlock, key)
					}
				}
				p.rwlock.RUnlock()
				if len(linksLackBlock) > 0 {
					event := &core.GetIsolateBlocksEvent{Hashes: linksLackBlock}
					p.blockEvent.Post(event)
				}
				lastTime = currentTime

				for _, block := range blockList {
					p.deleteIsolatedBlock(block)
				}
			}
			time.Sleep(time.Second)
		}
	}()
}

func (p *BlockPool) loop() {
	p.unverifiedBlocks.Push(p.mainChainI.GetTail().Hash)
	go func() {
		for {
			select {
			case ch := <-p.newAnnounceSub.Chan():
				if ev, ok := ch.Data.(*core.AnnounceEvent); ok {
					if inlocal := storage.HasBlock(p.db, ev.Hash); !inlocal {
						hashes := make([]common.Hash, 0)
						hashes = append(hashes, ev.Hash)
						event := &core.GetNetworkNewBlocksEvent{Hashes: hashes}
						p.blockEvent.Post(event)
						log.Debug("Post fetch block event", "hash", ev.Hash)
					}
				}
			case ch := <-p.queryUnverifySub.Chan():
				if ev, ok := ch.Data.(*core.GetUnverifyBlocksEvent); ok {
					ev.Hashes = p.SelectUnverifiedBlock(4)
					ev.Done <- struct{}{}
				}
			case hash := <-p.unverifiedAddChan:
				p.listLock.Lock()
				p.unverifiedBlocks.Push(hash)
				p.listLock.Unlock()

			case hash := <-p.unverifiedDelChan:
				p.listLock.Lock()
				p.unverifiedBlocks.Remove(hash)
				p.listLock.Unlock()
			}
		}
	}()

	p.TimedRequestForIsolatedBlocks()
	p.BlockProcessing()
}

func (p *BlockPool) addIsolatedBlock(block types.Block, links []common.Hash) bool {
	defer p.rwlock.Unlock()
	p.rwlock.Lock()

	log.Debug("begin addIsolatedBlock", "hash", block.GetHash().String())
	isolated := block.GetHash()
	if _, ok := p.IsolatedBlockMap[isolated]; ok {
		log.Debug("the Isolated block already exists")
		return false
	}

	newIsolatedBlock := &IsolatedBlock{links, []common.Hash{}, uint32(time.Now().Unix()), block.GetRlp()}
	needMeBlock, ok := p.lackBlockMap[isolated]
	if ok {
		newIsolatedBlock.LinkIt = append(newIsolatedBlock.LinkIt, needMeBlock.LinkIt...)
		delete(p.lackBlockMap, isolated)
	}

	p.IsolatedBlockMap[isolated] = newIsolatedBlock
	for _, link := range links {
		v, ok := p.IsolatedBlockMap[link] //if the ancestor of the block has already existed in orphan graph
		if ok {
			v.LinkIt = append(v.LinkIt, isolated) //update the ancestor's desendant list who is directly reference it
		} else { //marker the parent
			v, ok := p.lackBlockMap[link]
			if ok {
				v.LinkIt = append(v.LinkIt, isolated)
				if v.MinTime > block.GetTime() {
					v.MinTime = block.GetTime()
				}
			} else {
				p.lackBlockMap[link] = &lackBlock{[]common.Hash{isolated}, time.Now().Unix(), block.GetTime()}
			}
		}
	}

	log.Debug("end addIsolatedBlock", "hash", block.GetHash(), "IsolatedBlockMap len", len(p.IsolatedBlockMap), "lackBlockMap len", len(p.lackBlockMap))
	return true
}

type verifyMarker struct {
	block    types.Block
	verified bool
}

func (p *BlockPool) deleteIsolatedBlock(block types.Block) {
	defer p.rwlock.Unlock()
	p.rwlock.Lock()

	type DeleteIsolated struct {
		block  types.Block
		LinkIt []common.Hash
	}

	log.Debug("begin deleteIsolatedBlock", "hash", block.GetHash().String())
	count := 0
	blockHash := block.GetHash()
	v, ok := p.lackBlockMap[blockHash]
	if ok {
		delete(p.lackBlockMap, blockHash)
		currentList := []*DeleteIsolated{&DeleteIsolated{block, v.LinkIt}} // descendants
		var nextLayerList []*DeleteIsolated

		for len(currentList) > 0 {
			for _, deleteIsolated := range currentList { // process descendants by layer
				for _, hash := range deleteIsolated.LinkIt {
					isolated, ok := p.IsolatedBlockMap[hash]
					if !ok {
						log.Error("IsolatedBlockMap[hash] Exception")
						continue
					}
					var isFound bool
					for i := 0; i < len(isolated.Links); {
						if isolated.Links[i] == deleteIsolated.block.GetHash() {
							isolated.Links = append(isolated.Links[:i], isolated.Links[i+1:]...)
							isFound = true
						} else {
							i++
						}
					}

					if !isFound {
						log.Error("IsolatedBlockMap[hash] Links not found self")
					}
					if len(isolated.Links) == 0 {
						delete(p.IsolatedBlockMap, hash)
						// save block
						if fullBlock, err := types.BlockDecode(isolated.RLP); err == nil {
							p.deleteUnverifiedBlocks(fullBlock.GetLinks())
							p.addBlockLock.Lock()
							hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(fullBlock)
							if err == nil {
								log.Debug("ComputeCumulativeDiff finish", "hash", fullBlock.GetHash().String())
								p.saveBlock(fullBlock)
								p.addBlockLock.Unlock()
								if hasUpdateCumulativeDiff {
									p.mainChainI.UpdateTail(fullBlock)
								}
								nextLayerList = append(nextLayerList, &DeleteIsolated{fullBlock, isolated.LinkIt})

								log.Debug("Delete block from orphan graph", "hash", hash.String(), "IsolatedBlockMap len", len(p.IsolatedBlockMap), "lackBlockMap len", len(p.lackBlockMap))
								p.statisticsDeleteIsolatedBlock.Statistics(true)
								count++
							} else {
								p.addBlockLock.Unlock()
								log.Error("deleteIsolatedBlock ComputeCumulativeDiff failed", "block", hash.String(), "err", err)
							}
						} else {
							log.Error("Unserialize(UnRLP) failed", "block", hash.String(), "err", err)
						}
					}
				}
			}
			currentList = nextLayerList
			nextLayerList = []*DeleteIsolated{}
		}
	}

	log.Debug("end deleteIsolatedBlock", "count", count)
}

func (p *BlockPool) EnQueue(block types.Block) error {
	event := core.LocalNewBlocksEvent{Blocks: make([]types.Block, 0)}
	event.Blocks = append(event.Blocks, block)

	p.blockEvent.Post(&event)
	return nil
}

func (p *BlockPool) SyncAddBlock(block types.Block) error {
	p.blockChan <- block
	return nil
}

func (p *BlockPool) updateSyncTime(time uint64) {
	p.MaxSyncTimeLock.RLock()
	if p.MaxSyncTime < time {
		p.MaxSyncTimeLock.RUnlock()
		p.MaxSyncTimeLock.Lock()
		p.MaxSyncTime = time
		p.MaxSyncTimeLock.Unlock()
	} else {
		p.MaxSyncTimeLock.RUnlock()
	}
}

func (p *BlockPool) AddBlock(block types.Block, isRelay bool, isSync bool) error {
	log.Debug("begin AddBlock\n" + block.String())

	ok := storage.HasBlock(p.db, block.GetHash())
	if ok {
		log.Debug("the block has been added", "hash", block.GetHash().String())
		p.deleteIsolatedBlock(block)
		return fmt.Errorf("the block has been added")
	}

	isIsolated, err := p.linkCheckAndSave(block, isRelay, isSync)
	if err != nil {
		log.Error("linkCheckAndSave error" + err.Error())
	}

	log.Debug("addBlock finish", "hash", block.GetHash().String())

	p.statisticsAddBlock.Statistics((!isIsolated) && (err == nil))
	return err
}

func (p *BlockPool) linkCheckAndSave(block types.Block, isRelay bool, isSync bool) (bool, error) {
	var isIsolated bool
	var linksLackBlock []common.Hash

	for _, hash := range block.GetLinks() {
		linkBlock := storage.ReadBlock(p.db, hash)
		if linkBlock != nil {
			if linkBlock.GetTime() > block.GetTime() {
				log.Error("links time error", "block time", block.GetTime(), "link time", linkBlock.GetTime())
				return false, fmt.Errorf("links time error")
			}
		} else {
			isIsolated = true
			linksLackBlock = append(linksLackBlock, hash)
		}
	}

	if isSync {
		p.updateSyncTime(block.GetTime())
	}

	if isIsolated {
		log.Info("is a Isolated block", "hash", block.GetHash().String())
		p.addIsolatedBlock(block, linksLackBlock)
	} else {
		p.deleteUnverifiedBlocks(block.GetLinks())
		log.Debug("deleteUnverifiedBlocks finish", "hash", block.GetHash().String())
		p.addBlockLock.Lock()
		hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(block)
		if err != nil {
			p.addBlockLock.Unlock()
			return isIsolated, err
		}
		log.Debug("ComputeCumulativeDiff finish", "hash", block.GetHash().String())
		p.saveBlock(block)
		p.addBlockLock.Unlock()
		if hasUpdateCumulativeDiff {
			p.mainChainI.UpdateTail(block)
		}
		log.Debug("saveBlock finish", "hash", block.GetHash().String())
		p.deleteIsolatedBlock(block)

		if isRelay {
			log.Debug("Relay block", "hash", block.GetHash().String())
			event := &core.RelayBlocksEvent{Blocks: make([]types.Block, 0)}
			event.Blocks = append(event.Blocks, block)
			p.blockEvent.Post(event)
		}
	}

	return isIsolated, nil
}

func (p *BlockPool) saveBlock(block types.Block) {
	log.Debug("Save block", "hash", block.GetHash().String())
	storage.WriteBlock(p.db, block)
	p.addUnverifiedBlock(block.GetHash())
	p.sendBlockInfoToCenter(block)
}

func (p *BlockPool) verifyAncestor(ancestor types.Block) {
	p.deleteUnverifiedBlock(ancestor.GetHash())
}

func (p *BlockPool) verifyAncestors(ancestors []types.Block) {
	for _, ancestor := range ancestors {
		p.verifyAncestor(ancestor)
	}
}

func (p *BlockPool) deleteUnverifiedBlocks(hashSlice []common.Hash) {
	for _, hash := range hashSlice {
		p.deleteUnverifiedBlock(hash)
	}
}

func (p *BlockPool) addUnverifiedBlock(hash common.Hash) {
	select {
	case p.unverifiedAddChan <- hash:
		return
	}
}

func (p *BlockPool) deleteUnverifiedBlock(hash common.Hash) {
	select {
	case p.unverifiedDelChan <- hash:
		return
	}
}

func (p *BlockPool) SelectUnverifiedBlock(number int) []common.Hash {
	i := 0
	var links []common.Hash
	p.listLock.RLock()
	for itr, _ := p.unverifiedBlocks.Front(); itr != nil && i < number; itr = itr.Next() {
		if hash, ok := itr.Data().(common.Hash); ok {
			links = append(links, hash)
			//p.unverifiedBlocks.Remove(hash)
			i++
		} else {
			log.Error("error hash.(common.Hash): ", hash)
		}
	}
	p.listLock.RUnlock()
	return links
}

func (p *BlockPool) sendBlockInfoToCenter(block types.Block) {
	if p.mq == nil {
		return
	}
	var (
		ReceiverAddr string
		Amount       string
	)
	SenderAddr, _ := block.GetSender()
	var IsMiner string
	if block.GetType() == types.BlockTypeMiner {
		IsMiner = "1"
	} else {
		IsMiner = "0"
	}
	for _, out := range block.GetOuts() {
		ReceiverAddr += out.Receiver.String() + ","
		Amount += out.Amount.String() + ","
	}
	if len(ReceiverAddr) > 0 {
		ReceiverAddr = ReceiverAddr[:len(ReceiverAddr)-1]
	}
	if len(Amount) > 0 {
		Amount = Amount[:len(Amount)-1]
	}

	message := types.MQBlockInfo{
		BlockHash:      block.GetHash().String(),
		TransferDate:   strconv.FormatUint(block.GetTime(), 10),
		Amount:         Amount,
		GasLimit:       strconv.FormatUint(block.GetGasLimit(), 10),
		GasPrice:       block.GetGasPrice().String(),
		SenderAddr:     SenderAddr.String(),
		ReceiverAddr:   ReceiverAddr,
		IsMiner:        IsMiner,
		Difficulty:     block.GetDiff().String(),
		CumulativeDiff: block.GetCumulativeDiff().String(),
	}
	if err := p.mq.Publish("blockInfo", message); err != nil {
		log.Error(err.Error())
	}
}
