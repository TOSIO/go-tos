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

	IsolatedBlockMap map[common.Hash]IsolatedBlock
	lackBlockMap     map[common.Hash]lackBlock
	rwlock           sync.RWMutex

	maxQueueSize int

	newblockFeed *event.Feed
	blockEvent   *event.TypeMux

	newBlocksSub     *event.TypeMuxSubscription
	queryUnverifySub *event.TypeMuxSubscription

	newBlockAddChan   chan types.Block
	unverifiedAddChan chan common.Hash
	unverifiedDelChan chan common.Hash
	//unverifiedGetChan: make(chan unverifiedReq),
	unverifiedBlocks *container.UniqueList
	listLock         sync.RWMutex

	//UnverifiedBlockList *list.List
	statisticsObj statistics.Statistics

	mainChainI mainchain.MainChainI
}

func New(mainChain mainchain.MainChainI, chainDb tosdb.Database, feed *event.TypeMux) *BlockPool {
	pool := &BlockPool{
		mainChainI: mainChain,
		db:         chainDb,

		newblockFeed: &event.Feed{},
		emptyC:       make(chan struct{}, 1),
		blockChan:    make(chan types.Block, 8),

		IsolatedBlockMap: make(map[common.Hash]IsolatedBlock),
		lackBlockMap:     make(map[common.Hash]lackBlock),

		maxQueueSize: 10000,

		newBlockAddChan:   make(chan types.Block, 1000),
		unverifiedAddChan: make(chan common.Hash, 1000),
		unverifiedDelChan: make(chan common.Hash, 2000),

		blockEvent: feed,
		//newBlocksEvent: make(chan core.NewBlocksEvent, 8),
		//unverifiedGetChan: make(chan unverifiedReq),
	}
	pool.newBlocksSub = pool.blockEvent.Subscribe(&core.NewBlocksEvent{})
	pool.queryUnverifySub = pool.blockEvent.Subscribe(&core.GetUnverifyBlocksEvent{})
	pool.unverifiedBlocks = container.NewUniqueList(pool.maxQueueSize * 3)
	go pool.loop()
	return pool
}

func (p *BlockPool) SubscribeNewBlocksEvent(ev chan<- core.NewBlocksEvent) {
	p.newblockFeed.Subscribe(ev)
}

func (p *BlockPool) loop() {
	//UnverifiedBlockList = list.New()
	//UnverifiedBlockList.PushFront(genesis)
	p.unverifiedBlocks.Push(p.mainChainI.GetTail().Hash)
	go func() {
		var n int64 = 0
		currentTime := time.Now().UnixNano()
		for {
			select {
			case ch := <-p.newBlocksSub.Chan():
				if ev, ok := ch.Data.(*core.NewBlocksEvent); ok {
					go func(event *core.NewBlocksEvent) {
						for _, block := range event.Blocks {
							p.AddBlock(block)
							if n == 2000 {
								log.Warn("AddBlock Using time:" + fmt.Sprintf("%d", (time.Now().UnixNano()-currentTime)/n/1000))
								currentTime = time.Now().UnixNano()
								n = 0
							}
						}
					}(ev)
				}
			case ch := <-p.queryUnverifySub.Chan():
				if ev, ok := ch.Data.(*core.GetUnverifyBlocksEvent); ok {
					ev.Hashes = p.SelectUnverifiedBlock(4)
					ev.Done <- struct{}{}
				}
			case block := <-p.blockChan:
				go p.AddBlock(block)
				if n == 2000 {
					log.Warn("AddBlock Using time:" + fmt.Sprintf("%d", (time.Now().UnixNano()-currentTime)/n/1000))
					currentTime = time.Now().UnixNano()
					n = 0
				}
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

func (p *BlockPool) addIsolatedBlock(block types.Block, links []common.Hash) {
	defer p.rwlock.Unlock()
	p.rwlock.Lock()

	log.Debug("begin addIsolatedBlock", "hash", block.GetHash().String())
	isolated := block.GetHash()
	if _, ok := p.IsolatedBlockMap[isolated]; ok {
		//log.Warn("the Isolated block already exists")
		return
	}

	p.IsolatedBlockMap[isolated] = IsolatedBlock{links, []common.Hash{}, uint32(time.Now().Unix()), block.GetRlp()}
	for _, link := range links {
		v, ok := p.IsolatedBlockMap[link] //if the ancestor of the block has already existed in orphan graph
		if ok {
			v.LinkIt = append(v.LinkIt, isolated) //update the ancestor's desendant list who is directly reference it
			p.IsolatedBlockMap[link] = v
		} else { //marker the parent
			v, ok := p.lackBlockMap[link]
			if ok {
				v.LinkIt = append(v.LinkIt, isolated)
				p.lackBlockMap[link] = v
			} else {
				p.lackBlockMap[link] = lackBlock{[]common.Hash{isolated}, uint32(time.Now().Unix())}
			}
		}
	}

	lackBlock, ok := p.lackBlockMap[isolated]
	if ok {
		v := p.IsolatedBlockMap[isolated]
		v.LinkIt = append(v.LinkIt, lackBlock.LinkIt...)
		p.IsolatedBlockMap[isolated] = v
		delete(p.lackBlockMap, isolated)
	}
}

type verifyMarker struct {
	block    types.Block
	verified bool
}

func (p *BlockPool) deleteIsolatedBlock(block types.Block) {
	defer p.rwlock.Unlock()
	p.rwlock.Lock()

	log.Debug("begin deleteIsolatedBlock", "hash", block.GetHash().String())
	blockHash := block.GetHash()
	v, ok := p.lackBlockMap[blockHash]
	if ok {
		delete(p.lackBlockMap, blockHash)
		//save blcok
		currentList := v.LinkIt // descendants
		nextLayerList := []common.Hash{}
		ancestorCache := make(map[common.Hash]*verifyMarker)
		ancestorCache[blockHash] = &verifyMarker{block, false}

		for len(currentList) > 0 {
			for _, hash := range currentList { // process descendants by layer
				isolated := p.IsolatedBlockMap[hash]
				// verify ancestors
				for i, ancestor := range isolated.Links {
					if marker, ok := ancestorCache[ancestor]; ok && marker != nil {
						if !marker.verified {
							p.verifyAncestor(marker.block)
							marker.verified = true
						}
						isolated.Links = append(isolated.Links[:i], isolated.Links[i+1:]...)
					}
				}

				if len(isolated.Links) == 0 {
					// save block
					if fullBlock, err := types.BlockDecode(isolated.RLP); err == nil {
						ancestorCache[hash] = &verifyMarker{fullBlock, false}
						delete(p.IsolatedBlockMap, hash)
						hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(fullBlock)
						if err == nil {
							log.Debug("ComputeCumulativeDiff finish", "hash", fullBlock.GetHash().String())
							p.saveBlock(fullBlock)
							if hasUpdateCumulativeDiff {
								p.mainChainI.UpdateTail(fullBlock)
							}
							log.Debug("Delete block from orphan graph", "hash", blockHash.String(), "IsolatedBlockMap len", len(p.IsolatedBlockMap))
						} else {
							log.Error("deleteIsolatedBlock ComputeCumulativeDiff failed", "block", hash.String(), "err", err)
						}
					} else {
						log.Error("Unserialize(UnRLP) failed", "block", hash.String(), "err", err)
					}
				}
				nextLayerList = append(nextLayerList, isolated.LinkIt...)
			}
			currentList = nextLayerList
			nextLayerList = []common.Hash{}
		}
	}
}

func (p *BlockPool) EnQueue(block types.Block) error {
	event := core.NewBlocksEvent{Blocks: make([]types.Block, 0)}
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

func (p *BlockPool) AddBlock(block types.Block) error {
	log.Debug("begin AddBlock", "hash", block.GetHash().String(), "block", block)
	err := block.Validation()
	if err != nil {
		log.Error("the block Validation fail", "hash", block.GetHash().String())
		return fmt.Errorf("the block Validation fail")
	}

	ok := storage.HasBlock(p.db, block.GetHash())
	if ok {
		log.Error("the block has been added", "hash", block.GetHash().String())
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

	err = p.linkCheckAndSave(block)
	if err != nil {
		log.Error(err.Error())
	}
	//deleteIsolatedBlock(block)
	//go pm.RelayBlock(block.GetRlp())

	log.Debug("addBlock finish", "hash", block.GetHash().String())

	p.statisticsObj.Statistics()
	return err
}

func (p *BlockPool) linkCheckAndSave(block types.Block) error {
	var isIsolated bool
	var linkBlockIs []types.Block
	var linksLackBlock []common.Hash

	for _, hash := range block.GetLinks() {
		linkBlockEI := storage.ReadBlock(p.db, hash) //the 'EI' is empty interface logogram
		if linkBlockEI != nil {
			if linkBlockI, ok := linkBlockEI.(types.Block); ok {
				if linkBlockI.GetTime() > block.GetTime() {
					log.Error("links time error", "block time", block.GetTime(), "link time", linkBlockI.GetTime())
					return fmt.Errorf("links time error")
				} else {
					info, err := storage.ReadBlockMutableInfo(p.db, hash)
					if err != nil {
						log.Error("ReadBlockMutableInfo error", "hash", hash)
						return fmt.Errorf("ReadBlockMutableInfo error")
					}
					linkBlockI.SetMutableInfo(info)
					linkBlockIs = append(linkBlockIs, linkBlockI)
					//log.Trace("links time legal")
				}
			} else {
				log.Error("linkBlockEI assertion failure", "hash", hash)
				return fmt.Errorf("linkBlockEI assertion failure")
			}
		} else {
			isIsolated = true
			linksLackBlock = append(linksLackBlock, hash)
		}
	}

	if isIsolated {
		log.Warn("is a Isolated block", "hash", block.GetHash().String())
		p.addIsolatedBlock(block, linksLackBlock)
		/* for _, linkBlock := range linksLackBlock {
			//p.pm.GetBlock(linkBlock)

		} */
		log.Debug("Request ancestor", "hash", block.GetHash().String())
		event := &core.GetBlocksEvent{Hashes: linksLackBlock}
		p.blockEvent.Post(event)

	} else {
		//log.Trace("Verification passed")
		p.verifyAncestors(linkBlockIs)
		log.Debug("verifyAncestors finish", "hash", block.GetHash().String())
		hasUpdateCumulativeDiff, err := p.mainChainI.ComputeCumulativeDiff(block)
		if err != nil {
			return err
		}
		log.Debug("ComputeCumulativeDiff finish", "hash", block.GetHash().String())
		p.saveBlock(block)
		if hasUpdateCumulativeDiff {
			p.mainChainI.UpdateTail(block)
		}
		log.Debug("saveBlock finish", "hash", block.GetHash().String())
		p.deleteIsolatedBlock(block)

		log.Debug("Relay block", "hash", block.GetHash().String())
		event := &core.RelayBlocksEvent{Blocks: make([]types.Block, 0)}
		event.Blocks = append(event.Blocks, block)
		p.blockEvent.Post(event)
	}

	return nil
}

func (p *BlockPool) saveBlock(block types.Block) {
	log.Debug("Save block", "hash", block.GetHash().String())
	storage.WriteBlock(p.db, block)
	p.addUnverifiedBlock(block.GetHash())
	/* hash := block.GetHash()
	   addUnverifiedBlockList(hash) */
}

func (p *BlockPool) verifyAncestor(ancestor types.Block) {
	ancestor.SetStatus(ancestor.GetStatus() | types.BlockVerify)
	storage.WriteBlockMutableInfoRlp(p.db, ancestor.GetHash(), types.GetMutableRlp(ancestor.GetMutableInfo()))
	p.deleteUnverifiedBlock(ancestor.GetHash())
}

func (p *BlockPool) verifyAncestors(ancestors []types.Block) {
	for _, ancestor := range ancestors {
		p.verifyAncestor(ancestor)
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
			i++
		} else {
			log.Error("error hash.(common.Hash): ", hash)
		}
	}
	p.listLock.RUnlock()
	return links
}
