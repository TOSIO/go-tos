package manager

import (
	"fmt"
	"sync"
	"time"

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

var (
	pm        protocolManagerI
	db        tosdb.Database
	emptyC    = make(chan struct{}, 1)
	blockChan = make(chan types.Block, 8)

	IsolatedBlockMap = make(map[common.Hash]IsolatedBlock)
	lackBlockMap     = make(map[common.Hash]lackBlock)
	rwlock           sync.RWMutex

	maxQueueSize = 10000

	newBlockAddChan   = make(chan types.Block, 1000)
	unverifiedAddChan = make(chan common.Hash, 1000)
	unverifiedDelChan = make(chan common.Hash, 2000)
	//unverifiedGetChan: make(chan unverifiedReq),
	unverifiedBlocks = container.NewUniqueList(maxQueueSize * 3)
	listLock         sync.RWMutex

	//UnverifiedBlockList *list.List
	statisticsObj statistics.Statistics
)

func init() {
	//UnverifiedBlockList = list.New()
	//UnverifiedBlockList.PushFront(genesis)
	unverifiedBlocks.Push(GetMainBlockTail())
	go func() {
		var n int64 = 0
		currentTime := time.Now().UnixNano()
		for {
			select {
			case block := <-blockChan:
				go AddBlock(block)
				if n == 2000 {
					log.Warn("AddBlock Using time:" + fmt.Sprintf("%d", (time.Now().UnixNano()-currentTime)/n/1000))
					currentTime = time.Now().UnixNano()
					n = 0
				}
			case hash := <-unverifiedAddChan:
				listLock.Lock()
				unverifiedBlocks.Push(hash)
				listLock.Unlock()

			case hash := <-unverifiedDelChan:
				listLock.Lock()
				unverifiedBlocks.Remove(hash)
				listLock.Unlock()
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

func SetDB(chainDb tosdb.Database) {
	db = chainDb
}

func GetMainBlockTail() common.Hash {
	var hash common.Hash
	hash = core.GenesisHash
	return hash
}

func SetProtocolManager(protocolManager protocolManagerI) {
	pm = protocolManager
}

/*
func SetMemPool(mp *MemPool) {
	//mempool = mp
} */

func addIsolatedBlock(block types.Block, links []common.Hash) {
	defer rwlock.Unlock()
	rwlock.Lock()

	isolated := block.GetHash()
	if _, ok := IsolatedBlockMap[isolated]; ok {
		//log.Warn("the Isolated block already exists")
		return
	}

	IsolatedBlockMap[isolated] = IsolatedBlock{links, []common.Hash{}, uint32(time.Now().Unix()), block.GetRlp()}
	for _, link := range links {
		v, ok := IsolatedBlockMap[link] //if the ancestor of the block has already existed in orphan graph
		if ok {
			v.LinkIt = append(v.LinkIt, isolated) //update the ancestor's desendant list who is directly reference it
			IsolatedBlockMap[link] = v
		} else { //marker the parent
			v, ok := lackBlockMap[link]
			if ok {
				v.LinkIt = append(v.LinkIt, isolated)
				lackBlockMap[link] = v
			} else {
				lackBlockMap[link] = lackBlock{[]common.Hash{isolated}, uint32(time.Now().Unix())}
			}
		}
	}

	lackBlock, ok := lackBlockMap[isolated]
	if ok {
		v := IsolatedBlockMap[isolated]
		v.LinkIt = append(v.LinkIt, lackBlock.LinkIt...)
		IsolatedBlockMap[isolated] = v
		delete(lackBlockMap, isolated)
	}
}

type verifyMarker struct {
	block    types.Block
	verified bool
}

func deleteIsolatedBlock(block types.Block) {
	defer rwlock.Unlock()
	rwlock.Lock()

	blockHash := block.GetHash()
	v, ok := lackBlockMap[blockHash]
	if ok {
		log.Trace("Delete block from orphan graph", "block", blockHash.String())

		delete(lackBlockMap, blockHash)
		//save blcok
		currentList := v.LinkIt // descendants
		nextLayerList := []common.Hash{}
		ancestorCache := make(map[common.Hash]*verifyMarker)
		ancestorCache[blockHash] = &verifyMarker{block, false}

		for len(currentList) > 0 {
			for _, hash := range currentList { // process descendants by layer
				isolated := IsolatedBlockMap[hash]
				// verify ancestors
				for i, ancestor := range isolated.Links {
					if marker, ok := ancestorCache[ancestor]; ok && marker != nil {
						if !marker.verified {
							verifyAncestor(marker.block)
							marker.verified = true
						}
						isolated.Links = append(isolated.Links[:i], isolated.Links[i+1:]...)
					}
				}

				if len(isolated.Links) == 0 {
					// save block
					if fullBlock, err := types.BlockDecode(isolated.RLP); err == nil {
						ancestorCache[hash] = &verifyMarker{fullBlock, false}
						delete(IsolatedBlockMap, hash)
						saveBlock(fullBlock)
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

func SyncAddBlock(block types.Block) error {
	//emptyC <- struct{}{}
	//err := AddBlock(block)
	//<-emptyC
	blockChan <- block
	//time.Sleep(1)
	return nil
}

func AddBlock(block types.Block) error {
	err := block.Validation()
	if err != nil {
		log.Error("the block Validation fail")
		return fmt.Errorf("the block Validation fail")
	}

	ok := storage.HasBlock(db, block.GetHash())
	if ok {
		log.Error("the block has been added")
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

	err = linkCheckAndSave(block)

	//deleteIsolatedBlock(block)
	//go pm.RelayBlock(block.GetRlp())

	statisticsObj.Statistics()
	return nil
}

func linkCheckAndSave(block types.Block) error {
	var isIsolated bool
	var linkBlockIs []types.Block
	var linksLackBlock []common.Hash
	var haveLinkGenesis bool

	for _, hash := range block.GetLinks() {
		if hash == core.GenesisHash {
			haveLinkGenesis = true
		} else {
			linkBlockEI := storage.ReadBlock(db, hash) //the 'EI' is empty interface logogram
			if linkBlockEI != nil {
				if linkBlockI, ok := linkBlockEI.(types.Block); ok {
					if linkBlockI.GetTime() > block.GetTime() {
						log.Error("links time error")
						return fmt.Errorf("links time error")
					} else {
						linkBlockIs = append(linkBlockIs, linkBlockI)
						//log.Trace("links time legal")
					}
				} else {
					log.Error("linkBlockEI assertion failure")
					return fmt.Errorf("linkBlockEI assertion failure")
				}
			} else {
				isIsolated = true
				linksLackBlock = append(linksLackBlock, hash)
			}
		}
	}

	if isIsolated {
		log.Warn(block.GetHash().String() + "is a Isolated block")
		addIsolatedBlock(block, linksLackBlock)
		for _, linkBlock := range linksLackBlock {
			pm.GetBlock(linkBlock)
		}
	} else {
		//log.Trace("Verification passed")
		if haveLinkGenesis {
			deleteUnverifiedBlock(core.GenesisHash)
		}
		verifyAncestors(linkBlockIs)
		saveBlock(block)
		deleteIsolatedBlock(block)
		pm.RelayBlock(block.GetRlp())
	}
	return nil
}

func saveBlock(block types.Block) {
	storage.WriteBlock(db, block)
	addUnverifiedBlock(block.GetHash())
	/* hash := block.GetHash()
	   addUnverifiedBlockList(hash) */
}

func verifyAncestor(ancestor types.Block) {
	ancestor.SetStatus(ancestor.GetStatus() | types.BlockVerify)
	storage.WriteBlockMutableInfoRlp(db, ancestor.GetHash(), types.GetMutableRlp(ancestor.GetMutableInfo()))
	deleteUnverifiedBlock(ancestor.GetHash())
}

func verifyAncestors(ancestors []types.Block) {
	for _, ancestor := range ancestors {
		verifyAncestor(ancestor)
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
func addUnverifiedBlock(hash common.Hash) {
	select {
	case unverifiedAddChan <- hash:
		return
	}
}

func deleteUnverifiedBlock(hash common.Hash) {
	//p.tempSlice = append(p.tempSlice, hash)
	select {
	case unverifiedDelChan <- hash:
		return
		/* 	default:
		log.Trace("dropping request") */
	}
}

func SelectUnverifiedBlock(links []common.Hash) []common.Hash {
	i := 0
	listLock.RLock()
	for itr, _ := unverifiedBlocks.Front(); itr != nil && i < params.MaxLinksNum; itr = itr.Next() {
		if hash, ok := itr.Data().(common.Hash); ok {
			links = append(links, hash)
			i++
		} else {
			log.Error("error hash.(common.Hash): ", hash)
		}
	}
	listLock.RUnlock()
	return links
}
