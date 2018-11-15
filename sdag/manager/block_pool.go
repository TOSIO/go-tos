package manager

import (
	"fmt"
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
	LinkIt []common.Hash //Link it
	Time   uint32
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

	IsolatedBlockMap map[common.Hash]*IsolatedBlock
	lackBlockMap     map[common.Hash]*lackBlock
	rwlock           sync.RWMutex
	addBlockLock     sync.RWMutex

	maxQueueSize int

	newblockFeed *event.Feed
	blockEvent   *event.TypeMux

	newAnnounceSub      *event.TypeMuxSubscription
	localNewBlocksSub   *event.TypeMuxSubscription
	networkNewBlocksSub *event.TypeMuxSubscription
	syncResponseSub     *event.TypeMuxSubscription
	isolateResponseSub  *event.TypeMuxSubscription
	queryUnverifySub    *event.TypeMuxSubscription

	newBlockAddChan   chan types.Block
	unverifiedAddChan chan common.Hash
	unverifiedDelChan chan common.Hash
	//unverifiedGetChan: make(chan unverifiedReq),
	unverifiedBlocks *container.UniqueList
	listLock         sync.RWMutex

	//UnverifiedBlockList *list.List
	statisticsAddBlock            statistics.Statistics
	statisticsDeleteIsolatedBlock statistics.Statistics

	mainChainI mainchain.MainChainI
}

func New(mainChain mainchain.MainChainI, chainDb tosdb.Database, feed *event.TypeMux) *BlockPool {
	pool := &BlockPool{
		mainChainI: mainChain,
		db:         chainDb,

		newblockFeed: &event.Feed{},
		emptyC:       make(chan struct{}, 1),
		blockChan:    make(chan types.Block, 8),

		IsolatedBlockMap: make(map[common.Hash]*IsolatedBlock),
		lackBlockMap:     make(map[common.Hash]*lackBlock),

		maxQueueSize: 10000,

		newBlockAddChan:   make(chan types.Block, 1000),
		unverifiedAddChan: make(chan common.Hash, 1000),
		unverifiedDelChan: make(chan common.Hash, 2000),

		blockEvent: feed,
		//newBlocksEvent: make(chan core.NewBlocksEvent, 8),
		//unverifiedGetChan: make(chan unverifiedReq),
	}
	pool.newAnnounceSub = pool.blockEvent.Subscribe(&core.AnnounceEvent{})
	pool.localNewBlocksSub = pool.blockEvent.Subscribe(&core.LocalNewBlocksEvent{})
	pool.networkNewBlocksSub = pool.blockEvent.Subscribe(&core.NetworkNewBlocksEvent{})
	pool.syncResponseSub = pool.blockEvent.Subscribe(&core.SYNCResponseEvent{})
	pool.isolateResponseSub = pool.blockEvent.Subscribe(&core.IsolateResponseEvent{})
	pool.queryUnverifySub = pool.blockEvent.Subscribe(&core.GetUnverifyBlocksEvent{})
	pool.unverifiedBlocks = container.NewUniqueList(pool.maxQueueSize * 3)
	pool.statisticsAddBlock.Init("add block")
	pool.statisticsDeleteIsolatedBlock.Init("delete isolated block")
	go pool.loop()
	return pool
}

func (p *BlockPool) BlockProcessing() {
	localNewBlocks := make(chan types.Block, 10000)
	isolateResponse := make(chan types.Block, 10000)
	networkNewBlocks := make(chan types.Block, 10000)
	syncResponse := make(chan types.Block, 10000)
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
					p.AddBlock(block, true)
					continue
				default:
				}
				select {
				case block := <-isolateResponse:
					p.AddBlock(block, false)
					continue
				default:
				}
				select {
				case block := <-networkNewBlocks:
					p.AddBlock(block, true)
					continue
				default:
				}
				select {
				case block := <-syncResponse:
					p.AddBlock(block, false)
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
			currentTime = time.Now().Unix()
			if lastTime+params.TimePeriod/1000 < currentTime {
				var linksLackBlock []common.Hash
				p.rwlock.RLock()
				for key := range p.lackBlockMap {
					log.Debug("Request ancestor", "hash", key.String(), "lackBlockMap len", len(p.lackBlockMap))
					linksLackBlock = append(linksLackBlock, key)
				}
				p.rwlock.RUnlock()
				if len(linksLackBlock) > 0 {
					event := &core.GetIsolateBlocksEvent{Hashes: linksLackBlock}
					p.blockEvent.Post(event)
				}
				lastTime = currentTime
			}
			time.Sleep(time.Second)
		}
	}()
}

func (p *BlockPool) loop() {
	//UnverifiedBlockList = list.New()
	//UnverifiedBlockList.PushFront(genesis)
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
			//case ch := <-p.newBlocksSub.Chan():
			//	if ev, ok := ch.Data.(*core.NewBlocksEvent); ok {
			//		go func(event *core.NewBlocksEvent) {
			//			for _, block := range event.Blocks {
			//				p.AddBlock(block)
			//			}
			//		}(ev)
			//	}
			case ch := <-p.queryUnverifySub.Chan():
				if ev, ok := ch.Data.(*core.GetUnverifyBlocksEvent); ok {
					ev.Hashes = p.SelectUnverifiedBlock(4)
					ev.Done <- struct{}{}
				}
			//case block := <-p.blockChan:
			//	go p.AddBlock(block)
			case hash := <-p.unverifiedAddChan:
				p.listLock.Lock()
				p.unverifiedBlocks.Push(hash)
				p.listLock.Unlock()

			case hash := <-p.unverifiedDelChan:
				p.listLock.Lock()
				p.unverifiedBlocks.Remove(hash)
				p.listLock.Unlock()
				/* 	case req := <-p.unverifiedGetChan:
				i := 0
				for itr, _ := p.unverifiedBlocks.Front(); itr != nil && i < params.MaxLinksNum; itr = itr.Next() {
					if hash, ok := itr.Data().(common.Hash); ok {
						*req.links = append(*req.links, hash)
						//req.callback(hash, req.links)
						i++
					} else {
						log.Error("error hash.(common.Hash): ", hash)
					}
					req.done <- struct{}{}
				} */
			}
		}
	}()

	p.TimedRequestForIsolatedBlocks()
	p.BlockProcessing()
}

/* func SetDB(chainDb tosdb.Database) {
	db = chainDb
}



func SetProtocolManager(protocolManager protocolManagerI) {
	pm = protocolManager
}

func SetMainChain(mainChain mainchain.MainChainI) {
	mainChainI = mainChain
}
*/
/*
func SetMemPool(mp *MemPool) {
	//mempool = mp
} */

func (p *BlockPool) addIsolatedBlock(block types.Block, links []common.Hash) bool {
	defer p.rwlock.Unlock()
	p.rwlock.Lock()

	log.Debug("begin addIsolatedBlock", "hash", block.GetHash().String())
	isolated := block.GetHash()
	if _, ok := p.IsolatedBlockMap[isolated]; ok {
		//log.Warn("the Isolated block already exists")
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
			//p.IsolatedBlockMap[link] = v
		} else { //marker the parent
			v, ok := p.lackBlockMap[link]
			if ok {
				v.LinkIt = append(v.LinkIt, isolated)
				//p.lackBlockMap[link] = v
			} else {
				p.lackBlockMap[link] = &lackBlock{[]common.Hash{isolated}, uint32(time.Now().Unix())}
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
							hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(fullBlock)
							if err == nil {
								log.Debug("ComputeCumulativeDiff finish", "hash", fullBlock.GetHash().String())
								p.saveBlock(fullBlock)
								if hasUpdateCumulativeDiff {
									p.mainChainI.UpdateTail(fullBlock)
								}
								nextLayerList = append(nextLayerList, &DeleteIsolated{fullBlock, isolated.LinkIt})

								log.Debug("Delete block from orphan graph", "hash", hash.String(), "IsolatedBlockMap len", len(p.IsolatedBlockMap), "lackBlockMap len", len(p.lackBlockMap))
								p.statisticsDeleteIsolatedBlock.Statistics(true)
								count++
							} else {
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
	//emptyC <- struct{}{}
	//err := AddBlock(block)
	//<-emptyC
	p.blockChan <- block
	//time.Sleep(1)
	return nil
}

func (p *BlockPool) AddBlock(block types.Block, isRelay bool) error {
	log.Debug("begin AddBlock", "hash", block.GetHash().String(), "block", block)
	err := block.Validation()
	if err != nil {
		log.Error("the block Validation fail", "hash", block.GetHash().String())
		return fmt.Errorf("the block Validation fail")
	}

	ok := storage.HasBlock(p.db, block.GetHash())
	if ok {
		log.Debug("the block has been added", "hash", block.GetHash().String())
		p.deleteIsolatedBlock(block)
		return fmt.Errorf("the block has been added")
	} else {
		//log.Trace("Non-repeating block")
	}
	linksNumber := len(block.GetLinks())

	//log.Trace("links", "number", linksNumber)
	if linksNumber < 1 || linksNumber > params.MaxLinksNum {
		log.Error("the block linksNumber Exception.", "linksNumber", linksNumber)
		return fmt.Errorf("the block linksNumber =%d", linksNumber)
	}

	isIsolated, err := p.linkCheckAndSave(block, isRelay)
	if err != nil {
		log.Error("linkCheckAndSave error" + err.Error())
	}
	//deleteIsolatedBlock(block)
	//go pm.RelayBlock(block.GetRlp())

	log.Debug("addBlock finish", "hash", block.GetHash().String())

	p.statisticsAddBlock.Statistics((!isIsolated) && (err == nil))
	return err
}

func (p *BlockPool) linkCheckAndSave(block types.Block, isRelay bool) (bool, error) {
	var isIsolated bool
	//var linkBlockIs []types.Block
	var linksLackBlock []common.Hash

	for _, hash := range block.GetLinks() {
		linkBlockEI := storage.ReadBlock(p.db, hash) //the 'EI' is empty interface logogram
		if linkBlockEI != nil {
			if linkBlockI, ok := linkBlockEI.(types.Block); ok {
				if linkBlockI.GetTime() > block.GetTime() {
					log.Error("links time error", "block time", block.GetTime(), "link time", linkBlockI.GetTime())
					return false, fmt.Errorf("links time error")
				} else {
					//linkBlockIs = append(linkBlockIs, linkBlockI)
				}
			} else {
				log.Error("linkBlockEI assertion failure", "hash", hash)
				return false, fmt.Errorf("linkBlockEI assertion failure")
			}
		} else {
			isIsolated = true
			linksLackBlock = append(linksLackBlock, hash)
		}
	}

	if isIsolated {
		log.Info("is a Isolated block", "hash", block.GetHash().String())
		if p.addIsolatedBlock(block, linksLackBlock) {
			//log.Debug("Request ancestor", "hash", block.GetHash().String())
			//event := &core.GetBlocksEvent{Hashes: linksLackBlock}
			//p.blockEvent.Post(event)
		}
	} else {
		//log.Trace("Verification passed")
		p.deleteUnverifiedBlocks(block.GetLinks())
		log.Debug("deleteUnverifiedBlocks finish", "hash", block.GetHash().String())
		p.addBlockLock.Lock()
		hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(block)
		if err != nil {
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
	/* hash := block.GetHash()
	   addUnverifiedBlockList(hash) */
}

func (p *BlockPool) verifyAncestor(ancestor types.Block) {
	//ancestor.SetStatus(ancestor.GetStatus() | types.BlockVerify)
	//storage.WriteBlockMutableInfoRlp(p.db, ancestor.GetHash(), types.GetMutableRlp(ancestor.GetMutableInfo()))
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

/* func DeleteUnverifiedTransactionList(linkHash common.Hash) error {
	for e := UnverifiedBlockList.Front(); e != nil; e = e.Next() {
		if e.Value == linkHash {
			UnverifiedBlockList.Remove(e)
			return nil
		}
	}
	return fmt.Errorf("Not found the linkHash")
}
*/
func (p *BlockPool) addUnverifiedBlock(hash common.Hash) {
	select {
	case p.unverifiedAddChan <- hash:
		return
	}
}

func (p *BlockPool) deleteUnverifiedBlock(hash common.Hash) {
	//p.tempSlice = append(p.tempSlice, hash)
	select {
	case p.unverifiedDelChan <- hash:
		return
		/* 	default:
		log.Trace("dropping request") */
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
