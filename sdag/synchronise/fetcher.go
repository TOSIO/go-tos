package synchronise

import (
	"fmt"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/deckarep/golang-set"
)

type fetchTask struct {
	beginTime time.Time
	nodes     []string
	launch    int
	orphan    bool
}

type Fetcher struct {
	reqCh       chan common.Hash
	reqOrphanCh chan common.Hash
	resCh       chan core.Response

	flighting map[common.Hash]*fetchTask // key : hash, value : the peer who has been asked for fetch

	announced map[common.Hash]mapset.Set //key : hash, value : announcer

	announceCount map[string]int
	flightLock    sync.RWMutex
	announceLock  sync.RWMutex
	//knownBlocks map[string]mapset.Set // key : origin peer, value : blocks

	peerset core.PeerSet

	blockPoolEvent *event.TypeMux

	quit chan struct{}

	wg sync.WaitGroup
}

func NewFetcher(peers core.PeerSet, poolEvent *event.TypeMux) *Fetcher {
	fetcher := &Fetcher{
		reqCh:       make(chan common.Hash, 100),
		reqOrphanCh: make(chan common.Hash, 50000),
		resCh:       make(chan core.Response, 100),

		flighting:      make(map[common.Hash]*fetchTask),
		announced:      make(map[common.Hash]mapset.Set),
		announceCount:  make(map[string]int),
		quit:           make(chan struct{}),
		peerset:        peers,
		blockPoolEvent: poolEvent,
	}
	return fetcher
}

func (f *Fetcher) loop() {
	f.wg.Add(1)
	defer f.wg.Done()

	for i := 0; i < maxRoutineCount; i++ {
		go func(fetcher *Fetcher, ID int) {
			fetcher.wg.Add(1)
			defer fetcher.wg.Done()

		workloop:
			for {
				select {
				case hash, ok := <-f.reqCh:
					log.Debug(">> Fetching block", "hash", hash.String())
					if ok {
						f.fetch(hash, false)
					} else {
						break workloop
					}
				case hash, ok := <-f.reqOrphanCh:
					log.Debug(">> Fetching isolated block", "hash", hash.String())
					if ok {
						f.fetch(hash, true)
					} else {
						break workloop
					}
				case hash, ok := <-f.resCh:
					if ok {
						f.done(hash)
					} else {
						break workloop
					}

				}
			}
			log.Debug("Fetch worker exited", "ID", ID)
			return
		}(f, i)
	}
	detect := time.After(f.requestTTL())
	for {
		select {
		/* 		case hash := <-f.reqCh:
		   			log.Debug(">> Fetching block", "hash", hash.String())
		   			go f.fetch(hash)
		   		case hash := <-f.resCh:
					   go f.done(hash) */
		case <-detect:
			f.detectTimeout()
			detect = time.After(f.requestTTL())
		case <-f.quit:
			close(f.reqCh)
			close(f.resCh)
			log.Info("Fetcher was stopped")
			break
		}
	}
}

func (f *Fetcher) stop() {
	f.quit <- struct{}{}
	f.wg.Wait()
}

func (f *Fetcher) requestTTL() time.Duration {
	return time.Duration(60 * time.Second)
}

func (f *Fetcher) fetch(hash common.Hash, orphan bool) {
	f.flightLock.RLock()
	if _, ok := f.flighting[hash]; ok {
		f.flightLock.RUnlock()
		log.Trace("Already in fetching", "hash", hash.String(), "isolated", orphan)
		return
	}
	f.flightLock.RUnlock()

	origins := f.selectOrigins(hash)
	if len(origins) <= 0 {
		origins = f.randomSelectOrigins()
		if origins == nil {
			log.Trace("Not found data source(by random select)", "hash", hash.String(), "isolated", orphan)
			return
		}
	}

	var task fetchTask
	task.launch = 0
	task.nodes = make([]string, 0)
	task.beginTime = time.Now()
	task.orphan = orphan

	if len(origins) <= 0 {
		log.Trace("Not found data source", "hash", hash.String(), "isolated", orphan)
		return
	}
	for _, peer := range origins {
		peer.RequestBlock(hash)
		task.nodes = append(task.nodes, peer.NodeID())
	}
	f.flightLock.Lock()
	f.flighting[hash] = &task
	f.flightLock.Unlock()
}

func (f *Fetcher) detectTimeout() {
	event := &core.ErrorGetBlocksEvent{Requests: make([]core.BlockReq, 0, 100), Err: fmt.Errorf("timeout")}
	//del := make([]core.BlockReq, 0, 100)
	f.flightLock.Lock()
	for hash, task := range f.flighting {
		if time.Since(task.beginTime) >= f.requestTTL() {
			delete(f.flighting, hash)
			event.Requests = append(event.Requests, core.BlockReq{Hash: hash, Isolated: task.orphan})
		}
	}
	f.flightLock.Unlock()

	if len(event.Requests) > 0 {
		log.Debug("Post error event", "size", len(event.Requests))
		f.blockPoolEvent.Post(event)
		log.Debug("Post error event completed", "size", len(event.Requests))
	}
}

func (f *Fetcher) whoAnnounced(hash common.Hash) mapset.Set {
	//return nil
	f.announceLock.RLock()
	defer f.announceLock.RUnlock()

	var ret mapset.Set
	if announcers, ok := f.announced[hash]; ok {
		ret = announcers.Clone()
	}
	return ret
}

func (f *Fetcher) randomSelectOrigins() []core.Peer {
	if f.peerset != nil {
		peers := f.peerset.Peers()
		if len(peers) <= 0 {
			return nil
		}
		num := maxOringinPeersLimit
		if num > len(peers) {
			num = len(peers)
		}
		result := make([]core.Peer, 0, num)
		for _, peer := range peers {
			if peer != nil {
				result = append(result, peer)
				if len(result) >= num {
					return result
				}
			}
		}
	}
	return nil
}

func (f *Fetcher) selectOrigins(hash common.Hash) []core.Peer {
	//var err error
	result := make([]core.Peer, 0)
	announcers := f.whoAnnounced(hash)
	if announcers != nil {
		itr := announcers.Iterator()
		for e := range itr.C {
			nodeID := e.(string)
			peer := f.peerset.FindPeer(nodeID)
			if peer != nil {
				result = append(result, peer)
			}
		}
	}
	return result
}

func (f *Fetcher) MarkAnnounced(hash common.Hash, nodeId string) {
	f.announceLock.Lock()
	defer f.announceLock.Unlock()

	if v, ok := f.announced[hash]; ok {
		v.Add(nodeId)
	} else {
		nodes := make([]string, 0)
		nodes = append(nodes, nodeId)
		f.announced[hash] = mapset.NewSet()
		f.announced[hash].Add(nodeId)
		f.announceCount[nodeId]++
	}
}

func (f *Fetcher) MarkFlighting(hashes []common.Hash, nodeID string) {
	f.flightLock.Lock()
	for _, hash := range hashes {
		var task fetchTask
		task.launch = 0
		task.nodes = make([]string, 0)
		task.beginTime = time.Now()
		task.nodes = append(task.nodes, nodeID)
		f.flighting[hash] = &task
	}
	f.flightLock.Unlock()
}

func (f *Fetcher) UnMarkFlighting(hashes []common.Hash) {
	f.flightLock.Lock()
	for _, hash := range hashes {
		f.remove(hash)
	}
	f.flightLock.Unlock()
}
func (f *Fetcher) remove(hash common.Hash) {
	delete(f.flighting, hash)
}

func (f *Fetcher) done(response core.Response) {
	f.processResponse(response)
}

func (f *Fetcher) processResponse(response core.Response) {
	//log.Info("Process block response")
	/* s.blockQueueLock.Lock()
	defer s.blockQueueLock.Unlock() */

	/* f.flightLock.Lock()
	defer f.flightLock.Lock() */
	//delHashes := make([]common.Hash, 0)
	newblockEvent := &core.NetworkNewBlocksEvent{Blocks: make([]types.Block, 0, 1)}
	isolateEvent := &core.IsolateResponseEvent{Blocks: make([]types.Block, 0, 1)}
	if packet, ok := response.(*NewBlockPacket); ok {
		f.flightLock.Lock()
		for _, item := range packet.blocks {
			if block, err := types.BlockDecode(item); err == nil {
				if task, ok := f.flighting[block.GetHash()]; ok {
					if task.orphan {
						isolateEvent.Blocks = append(isolateEvent.Blocks, block)
					} else {
						newblockEvent.Blocks = append(newblockEvent.Blocks, block)
					}
					delete(f.flighting, block.GetHash())
					log.Debug("Post block event to mempool", "hash", block.GetHash().String())
				} else {
					log.Debug("Found block response is not in flighting", "hash", block.GetHash())
				}
				//f.remove(block.GetHash())
				//delHashes = append(delHashes, block.GetHash())

			} else {
				log.Error("Error post block event to mempool", "err", err, "packet", item)
			}
		}
		f.flightLock.Unlock()
	}
	//f.UnMarkFlighting(delHashes)
	if len(newblockEvent.Blocks) > 0 {
		f.blockPoolEvent.Post(newblockEvent)
	}
	if len(isolateEvent.Blocks) > 0 {
		f.blockPoolEvent.Post(isolateEvent)
	}
}

func (f *Fetcher) AsyncRequestBlock(hash common.Hash) error {
	select {
	case f.reqCh <- hash:
		return nil
	default:
		log.Debug("Error push to queue,drop it", "hash", hash, "err", errQueueFull)
		return errQueueFull
	}
	//s.blockQueueLock.Lock()
	//s.blockReqQueue[hash] = ""
	//s.blockQueueLock.Unlock()

	//s.blockReqCh <- struct{}{}
}

func (f *Fetcher) AsyncRequestOrphanBlock(hash common.Hash) error {
	select {
	case f.reqOrphanCh <- hash:
		return nil
	default:
		log.Debug("Error push to queue,drop it", "hash", hash, "err", errQueueFull)
		return errQueueFull
	}
	//s.blockQueueLock.Lock()
	//s.blockReqQueue[hash] = ""
	//s.blockQueueLock.Unlock()

	//s.blockReqCh <- struct{}{}
}
