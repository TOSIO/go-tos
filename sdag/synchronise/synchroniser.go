package synchronise

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TOSIO/go-tos/sdag/core/protocol"

	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/utils"

	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
)

var maxRoutineCount int = 100
var maxSYNCCapLimit = 10000
var maxRetryCount = 3
var maxOringinPeersLimit = 25
var maxAnnouceLimitPeer = 10000

var (
	errNoSyncActive   = errors.New("no sync active")
	errInternal       = errors.New("internal error")
	errSendMsgTimeout = errors.New("send message timeout")
	errQueueFull      = errors.New("queue is full")
)

type timesliceHash struct {
	timeslice uint64
	hashes    []common.Hash
}

type Synchroniser struct {
	peerset core.PeerSet

	mainChain  mainchain.MainChainI
	blkstorage BlockStorageI

	netCh   chan int
	netFeed *event.Feed
	netSub  event.Subscription

	blockPoolEvent *event.TypeMux
	syncEvent      *event.TypeMux
	newTaskSub     *event.TypeMuxSubscription

	fetcher *Fetcher
	relayer *Relayer

	syncing          int32
	oneSliceSyncDone chan uint64

	peerSliceCh   chan core.Response
	blockhashesCh chan core.Response
	blocksCh      chan core.Response

	syncreqCh         chan core.Request
	blockresCh        chan core.Response
	blockresAckCh     chan core.Response
	locatorBlockresCh chan core.Response
	//newTask chan core.NewSYNCTask

	blockReqCh chan struct{}

	netStatus int
	/* 	blockReqQueue      map[common.Hash]string
	   	blockUnfinishQueue map[common.Hash]string
	   	blockQueueLock     sync.RWMutex // Lock to protect the cancel channel and peer in delivers
	*/
	done     chan struct{}
	cancelCh chan struct{} // Channel to cancel mid-flight syncs
	quitCh   chan struct{} // Channel to cancel mid-flight syncs

	sliceCache map[string]*timesliceHash
	cachelock  sync.RWMutex
	//syncResultCh chan error
	genesisTimeslice uint64
	genesis          common.Hash
	lastSYStimeslice uint64         //last syschronise time
	cancelLock       sync.RWMutex   // Lock to protect the cancel channel and peer in delivers
	wg               sync.WaitGroup // for shutdown sync
}

func NewSynchroinser(ps core.PeerSet, mc mainchain.MainChainI, bs BlockStorageI, feed *event.Feed, poolEvent *event.TypeMux, event *event.TypeMux /* , resultCh chan error */) (*Synchroniser, error) {
	syncer := &Synchroniser{peerset: ps,
		mainChain:      mc,
		blkstorage:     bs,
		blockPoolEvent: poolEvent,
		syncEvent:      event,
		netFeed:        feed,
		syncing:        0,
		netStatus:      core.NETWORK_CLOSED}
	syncer.oneSliceSyncDone = make(chan uint64)
	syncer.peerSliceCh = make(chan core.Response)
	syncer.blockhashesCh = make(chan core.Response)
	syncer.blocksCh = make(chan core.Response)
	syncer.blockReqCh = make(chan struct{})

	syncer.blockresCh = make(chan core.Response, 15000)
	syncer.blockresAckCh = make(chan core.Response, 15000)
	syncer.locatorBlockresCh = make(chan core.Response, 10000)

	syncer.syncreqCh = make(chan core.Request)
	syncer.sliceCache = make(map[string]*timesliceHash)

	syncer.done = make(chan struct{})
	syncer.cancelCh = make(chan struct{})
	syncer.quitCh = make(chan struct{})
	/* syncer.syncResultCh = resultCh */
	syncer.fetcher = NewFetcher(ps, poolEvent)
	syncer.relayer = NewRelayer(syncer.fetcher, ps)

	syncer.netCh = make(chan int)
	syncer.newTaskSub = syncer.syncEvent.Subscribe(&core.NewSYNCTask{})
	//syncer.netSub = syncer.netFeed.Subscribe(syncer.netCh)

	return syncer, nil
}

func (s *Synchroniser) Start() error {
	if genesis, err := s.mainChain.GetGenesisHash(); err == nil {
		s.genesis = genesis
		if genesisBlock := s.blkstorage.GetBlock(genesis); genesisBlock != nil {
			s.genesisTimeslice = utils.GetMainTime(genesisBlock.GetTime())
			s.lastSYStimeslice = s.genesisTimeslice
		}
	}
	go s.fetcher.loop()
	go s.relayer.loop()
	go s.loop()
	return nil
}

func (s *Synchroniser) Stop() {
	s.quitCh <- struct{}{}
	s.cancelCh <- struct{}{}
	s.fetcher.stop()
	s.relayer.stop()
	s.wg.Wait()
}

func (s *Synchroniser) schedule(tasks map[string]*core.NewSYNCTask, idle *bool) {
	nowTimeslice := utils.GetMainTime(utils.GetTOSTimeStamp())
	lastTempMBTimeslice := s.mainChain.GetLastTempMainBlkSlice()
	if lastTempMBTimeslice >= nowTimeslice {
		//s.netFeed.Send(core.SDAGSYNC_COMPLETED)
		log.Info("Synchronise is completed", "lastTS", s.mainChain.GetLastTempMainBlkSlice(), "nowTS", nowTimeslice)
		return
	}

	if atomic.LoadInt32(&s.syncing) != 0 {
		log.Debug("Synchroniser is in synchronizing")
		return
	}
	if lastTempMBTimeslice < s.lastSYStimeslice {
		log.Debug("timeslice", "lastTempMBTimeslice:", lastTempMBTimeslice, "lastSYStimeslice:", s.lastSYStimeslice)
		log.Debug("Synchroniser is not completed")
		return
	}

	target := ""
	var origin *core.NewSYNCTask
	var peer core.Peer
	for peer == nil && len(tasks) > 0 {
		maxMainBlockNum := uint64(0)
		for i, task := range tasks {
			if task.LastMainBlockNum > maxMainBlockNum {
				maxMainBlockNum = task.LastMainBlockNum
				target = i
			}
		}
		origin = tasks[target]
		delete(tasks, target)
		peer = s.peerset.FindPeer(target)
	}
	if peer == nil {
		return
	}
	if target != "" && s.mainChain.GetMainTail().CumulativeDiff.Cmp(&origin.LastCumulatedDiff) < 0 &&
		/* origin.LastMainBlockNum > s.mainChain.GetMainTail().Number && */
		origin.LastTempMBTimeslice > s.mainChain.GetLastTempMainBlkSlice()+3 {
		/* 	beginTimeslice := s.mainChain.GetLastTempMainBlkSlice() - 32
		if beginTimeslice < 0 || beginTimeslice < s.genesisTimeslice {
			beginTimeslice = s.genesisTimeslice
			log.Debug("Adjust the begin timeslice", "begin", beginTimeslice)
		} */
		*idle = false
		log.Debug("Post SYNC-SYNCING feed")
		s.netFeed.Send(core.SDAGSYNC_SYNCING)
		log.Debug("Post SYNC-SYNCING feed End")
		//go s.synchroinise(peer, beginTimeslice, origin.LastTempMBTimeslice, origin.LastMainBlockNum)
		atomic.StoreInt32(&s.syncing, 1)
		go s.synchroiniseV2(peer)
	} else {
		for key, _ := range tasks {
			delete(tasks, key)
		}
	}
}

func (self *Synchroniser) listenInbound() {
	for i := 0; i < maxRoutineCount; i++ {
		go func(s *Synchroniser, ID int) {
			s.wg.Add(1)
			defer s.wg.Done()
		workloop:
			for {
				select {
				case req, ok := <-s.syncreqCh:
					if ok {
						s.handleSYNCBlockRequest(req)
					} else {
						break workloop
					}

				case packet, ok := <-s.blockresAckCh:
					if ok {
						s.handleSYNCBlockResponseACK(packet)
					} else {
						break workloop
					}
				}
			}
			log.Debug("Synchronise inbound-worker exited", "ID", ID)
		}(self, i)
	}
}

func (s *Synchroniser) loop() {
	//var lastNetUnavilableTime time.Time
	s.wg.Add(1)
	defer s.wg.Done()

	queuedTask := make(map[string]*core.NewSYNCTask)
	discharg := func(resCh chan core.Response /* , resackCh chan core.Response */) {
		select {
		/* case <-resackCh:
		log.Debug("Discharge ACK") */
		case <-resCh:
			log.Debug("Discharge SYNC-BLOCK-RESPONSE")
		default:

		}
	}
	s.listenInbound()
	idle := true
	for {
		s.schedule(queuedTask, &idle)
		ticker := time.NewTicker(time.Second * 60)
		if idle {
			discharg(s.blockresCh)
		}
		select {
		case newTask := <-s.newTaskSub.Chan():
			if task, ok := newTask.Data.(*core.NewSYNCTask); ok {
				queuedTask[task.NodeID] = task
			}
		case <-s.done:
			idle = true
			continue
		case <-s.quitCh:
			close(s.syncreqCh)
			close(s.blockresAckCh)
			return
		case <-ticker.C:
			continue
		}
	}
}

func (s *Synchroniser) synchroiniseV2(peer core.Peer) {
	s.wg.Add(1)
	clear := func() {
		s.wg.Done()
		s.done <- struct{}{}
		// time.Sleep(time.Duration(180) * time.Second)
		atomic.StoreInt32(&s.syncing, 0)

	}
	defer clear()

	var (
		waitTimeout   <-chan time.Time
		stat          core.SYNCStatusEvent
		lastAck       *protocol.TimesliceIndex
		forkTimeslice uint64
	)
	stat = core.SYNCStatusEvent{
		Progress:          core.SYNC_READY,
		BeginTS:           0,
		EndTS:             0,
		CurTS:             0,
		AccumulateSYNCNum: 0,
		BeginTime:         time.Now(),
		CurOrigin:         peer.NodeID() + "[" + peer.Address() + "]",
		//TriedOrigin:       make([]string, 0)
	}
	s.syncEvent.Post(stat)

	if err := peer.SendGetlocatorRequest(); err != nil {
		log.Debug("Error send get-locator-request-message", "nodeid ", peer.NodeID(), "err", err)
		return
	} else {
		log.Debug("Send get-locator-request-message OK")

	}

	ticker := time.NewTicker(60 * time.Second)
	retry := 0
	retryGetlocator := 0
	postErrStat := func(status *core.SYNCStatusEvent, err error) {
		status.Err = err
		status.EndTime = time.Now()
		status.Progress = core.SYNC_ERROR
		s.syncEvent.Post(stat)
	}

loop:
	for {
		select {
		case response := <-s.locatorBlockresCh:
			//var locatorResp []protocol.MainChainSample
			ticker.Stop()
			retryGetlocator = 0
			log.Debug("Reset retryGetlocator")
			forkTimeslice = uint64(0)
			if locatorPacket, ok := response.(*LocatorPacket); ok {
				log.Debug("decode response msg", "locatorPacket", locatorPacket.response, "size", len(locatorPacket.response))
				numberEnd := s.mainChain.GetMainTail().Number
				log.Debug("Local current end", "tailnumber", numberEnd)

			internalloop:

				for _, sample := range locatorPacket.response {
					if sample.Number > numberEnd {
						continue
					}
					if hash, err := s.blkstorage.GetMainBlock(sample.Number); err == nil {
						if hash == sample.Hash {
							block := s.blkstorage.GetBlock(sample.Hash)
							if block != nil {
								forkTimeslice = utils.GetMainTime(block.GetTime())
								break internalloop
							} else {
								log.Debug("get block is error,can not to syschronsize")
								return
							}
						}
					}
				}

				forkTimeslice -= 2
				if forkTimeslice < s.genesisTimeslice {
					forkTimeslice = s.genesisTimeslice
					log.Debug("Adjust the begin timeslice", "begin", forkTimeslice)
				}

				stat.BeginTS = forkTimeslice

				if err := peer.SendSYNCBlockRequest(forkTimeslice, 0); err != nil {
					log.Debug("Error send SYNC-request-message", "beginTS", forkTimeslice, "err", err)
					return
				} else {
					log.Debug("Send SYNC-request-message OK", "timeslice", forkTimeslice)
				}
				waitTimeout = time.After(s.requestTTL())

				stat.CurTS = forkTimeslice
				stat.Err = nil
				stat.Progress = core.SYNC_SYNCING
				s.syncEvent.Post(stat)
				break
			} else {
				fmt.Printf("not locate begintimeslice")
			}

		case packet := <-s.blockresCh:
			log.Debug("Reset retry")
			if lastTSIndex, end, err := s.handleSYNCBlockResponse(packet, &stat); err == nil {
				retry = 0
				if end {
					log.Debug("Meet ending", "nodeID", packet.NodeID(), "endTimeslice", lastTSIndex.Timeslice)
					stat.Err = nil
					stat.CurTS = lastTSIndex.Timeslice
					stat.EndTime = time.Now()
					stat.Progress = core.SYNC_END
					s.lastSYStimeslice = lastTSIndex.Timeslice
					s.syncEvent.Post(stat)
					log.Debug("Post SYNC-COMPLETED feed")
					s.netFeed.Send(core.SDAGSYNC_COMPLETED)
					log.Debug("Post SYNC-COMPLETED feed End")
					break loop
				} else {
					//lastAck = lastTSIndex.Timeslice
					log.Debug("Post progress", "nodeID", packet.NodeID(), "timeslice", lastTSIndex.Timeslice, "index", lastTSIndex.Index)
					lastAck = lastTSIndex
					stat.CurTS = lastAck.Timeslice
					stat.Index = lastAck.Index
					s.lastSYStimeslice = lastTSIndex.Timeslice
					log.Debug("lastSYStimeslice", "lastSYStimeslice:", s.lastSYStimeslice)
					stat.Err = nil
					stat.Progress = core.SYNC_SYNCING
					s.syncEvent.Post(stat)
					log.Debug("Post progress completed", "nodeID", packet.NodeID(), "timeslice", lastTSIndex.Timeslice, "index", lastTSIndex.Index)
					if err = peer.SendSYNCBlockResponseACK(lastTSIndex.Timeslice, lastTSIndex.Index); err != nil {
						log.Debug("Error send SYNC-BLOCK-RESPONSE-ACK message", "nodeID", packet.NodeID(), "timeslice", lastTSIndex.Timeslice, "index", lastTSIndex.Index)
						postErrStat(&stat, err)
						break loop
					}
					waitTimeout = time.After(s.requestTTL())
				}
			} else {
				log.Debug("Error hanlde SYNC-BLOCK-RESPONSE", "nodeID", packet.NodeID())
			}
		case <-waitTimeout:
			if retry >= maxRetryCount {
				postErrStat(&stat, fmt.Errorf("timeout"))
				log.Debug("Stop to retry send SYNC-BLOCK-*** request")
				break loop
			}
			retry++
			log.Debug("Retry send SYNC-BLOCK-*** request", "retry", retry)
			if lastAck != nil {
				if err := peer.SendSYNCBlockResponseACK(lastAck.Timeslice, lastAck.Index); err != nil {
					log.Debug("Error send SYNC-BLOCK-RESPONSE-ACK message", "timeslice", lastAck.Timeslice, "index", lastAck.Index)
					postErrStat(&stat, err)
					break loop
				}
			} else {
				if err := peer.SendSYNCBlockRequest(forkTimeslice, 0); err != nil {
					log.Debug("Error send SYNC-request-message", "beginTS", forkTimeslice, "err", err)
					postErrStat(&stat, err)
					break loop
				} else {
					log.Debug("Send request-message OK", "timeslice", forkTimeslice)
				}
			}
			waitTimeout = time.After(s.requestTTL())
		case <-s.cancelCh:
			break loop

		case <-ticker.C:
			log.Debug("Get lcoator Packege is time out")
			if retryGetlocator >= maxRetryCount {
				postErrStat(&stat, fmt.Errorf("timeout"))
				log.Debug("Stop to retry send-get-locator  request")

				break loop
			}
			retryGetlocator++
			log.Debug("Retry send get-locator-request", "retry", retryGetlocator)
			if err := peer.SendGetlocatorRequest(); err != nil {
				log.Debug("Error send get-locator-request-message", "nodeid ", peer.NodeID(), "err", err)
			} else {
				log.Debug("Send get-locator-request-message OK")
			}
			ticker = time.NewTicker(60 * time.Second)
		}
	}

}

func (s *Synchroniser) handleSYNCBlockResponse(packet core.Response, stat *core.SYNCStatusEvent) (*protocol.TimesliceIndex, bool, error) {
	log.Debug("Handle SYNC-BLOCK-RESPONSE", "size", packet.ItemCount())
	maxTSIndex := protocol.TimesliceIndex{Timeslice: 0, Index: 0}
	end := false
	if res, ok := packet.(*SYNCBlockRespPacket); ok {
		if res.response != nil {
			end = res.response.End
			stat.EndTS = res.response.CurEndTimeslice
			newblockEvent := &core.SYNCResponseEvent{Blocks: make([]types.Block, 0)}
			log.Debug("Response dump", "end", end, "curEndPoint", stat.EndTS)
			for _, tsblocks := range res.response.TSBlocks {
				if maxTSIndex.Timeslice < tsblocks.TSIndex.Timeslice {
					maxTSIndex.Timeslice = tsblocks.TSIndex.Timeslice
					maxTSIndex.Index = tsblocks.TSIndex.Index
				}
				//hashes := make([]common.Hash, 0)
				for _, blk := range tsblocks.Blocks {
					//s.mempool.EnQueue(blk)
					if block, err := types.BlockDecode(blk); err == nil {
						newblockEvent.Blocks = append(newblockEvent.Blocks, block)
						//hashes = append(hashes, block.GetHash())
					}
				}
				//log.Debug("process response 1")
				//s.fetcher.UnMarkFlighting(hashes)
				//log.Debug("process response 2")

				//if tsblocks.Blocks
			}
			stat.AccumulateSYNCNum += uint64(len(newblockEvent.Blocks))
			if len(newblockEvent.Blocks) > 0 {
				log.Debug("Post newblock event to pool", "nodeID", res.NodeID(), "maxTimeslice", maxTSIndex.Timeslice, "index", maxTSIndex.Index)
				s.blockPoolEvent.Post(newblockEvent)
				log.Debug("Post newblock event to pool completed", "nodeID", res.NodeID(), "maxTimeslice", maxTSIndex.Timeslice, "index", maxTSIndex.Index)
			}
			log.Debug("Handle SYNC-BLOCK-RESPONSE response completed", "nodeID", res.NodeID(), "maxTimeslice", maxTSIndex.Timeslice, "index", maxTSIndex.Index, "size", len(newblockEvent.Blocks))
		} else {
			return nil, false, fmt.Errorf("response is invalid")
		}
	} else {
		return nil, false, fmt.Errorf("invalid type")
	}
	return &maxTSIndex, end, nil
}

func (s *Synchroniser) handleSYNCBlockRequest(packet core.Request) {
	if req, ok := packet.(*SYNCBlockReqPacket); ok {
		response := &protocol.SYNCBlockResponseACK{ConfirmPoint: *req.beginPoint}
		fake := &SYNCBlockResACKPacket{peerID: req.peerID, response: response}
		s.handleSYNCBlockResponseACK(fake)
	}

}

func (s *Synchroniser) handleSYNCBlockResponseACK(packet core.Response) error {
	/* forwardPack := func(syncer *Synchroniser, ID string, timeslice uint64, out *protocol.SYNCBlockResponse, cnt int) uint64 {
		//timeslice := ack.response.ConfirmPoint.Timeslice
		for cnt < maxSYNCCapLimit {
			timeslice++
			tsblocks := protocol.TimesliceBlocks{}
			tsblocks.TSIndex.Index = 0
			if hashes, err := syncer.blkstorage.GetBlockHashByTmSlice(timeslice); err == nil {
				tsblocks.TSIndex.Timeslice = timeslice

				if len(hashes) <= maxSYNCCapLimit-cnt {
					if blocks, err := syncer.blkstorage.GetBlocks(hashes); err == nil {
						tsblocks.Blocks = blocks
						out.TSBlocks = append(out.TSBlocks, tsblocks)
						cnt += len(hashes)
						continue
					}
				} else {
					syncer.cachelock.Lock()
					syncer.sliceCache[ID] = hashes
					syncer.cachelock.Unlock()
					if blocks, err := syncer.blkstorage.GetBlocks(hashes[0 : maxSYNCCapLimit-cnt]); err == nil {
						tsblocks.Blocks = blocks
						out.TSBlocks = append(out.TSBlocks, tsblocks)
						break
					}
				}
			} else {
				log.Debug("Error get local block-hash", "timeslice", timeslice)
			}
		}
		return timeslice
	} */
	//	log.Debug("handleSYNCBlockResponseACK 1")
	if ack, ok := packet.(*SYNCBlockResACKPacket); ok {
		curEndPoint := utils.GetMainTime(s.mainChain.GetMainTail().Time)
		//maxCap := 3000
		nodeID := ack.NodeID()
		count := 0
		response := protocol.SYNCBlockResponse{TSBlocks: make([]*protocol.TimesliceBlocks, 0)}
		response.CurEndTimeslice = curEndPoint
		endTimeslice :=
			ack.response.ConfirmPoint.Timeslice

		if ack.response != nil {
			remain := false
			var cache *timesliceHash
			var existed bool
			s.cachelock.RLock()
			cache, existed = s.sliceCache[nodeID]
			s.cachelock.RUnlock()

			if existed &&
				cache != nil &&
				cache.timeslice == ack.response.ConfirmPoint.Timeslice {

				pos := int(ack.response.ConfirmPoint.Index)

				if len(cache.hashes) > pos+1 { // handle the remain block since last sync
					tsblocks := &protocol.TimesliceBlocks{}
					tsblocks.TSIndex.Timeslice = ack.response.ConfirmPoint.Timeslice
					tsblocks.TSIndex.Index = ack.response.ConfirmPoint.Index
					tsblocks.Blocks = make([][]byte, 0)
					endTimeslice = ack.response.ConfirmPoint.Timeslice

					for pos++; count < maxSYNCCapLimit && pos < len(cache.hashes); pos++ {
						block := s.blkstorage.GetBlock(cache.hashes[pos])
						if block != nil {
							tsblocks.TSIndex.Index = uint(pos)
							tsblocks.Blocks = append(tsblocks.Blocks, block.GetRlp())
							count++
						} else {
							log.Debug("Error load block", "nodeID", nodeID, "timeslice", cache.timeslice, "hash", cache.hashes[pos].String())
						}
					}
					log.Debug("Handle packing(remain)", "nodeID", nodeID, "timeslice", cache.timeslice,
						"total", len(cache.hashes), "curPos", pos-1, "curPacking", len(tsblocks.Blocks), "remain", len(cache.hashes)-pos-1)

					//log.Debug("Make SYNC-BLOCK-RESPONSE(remain)", "nodeID", nodeID, "timeslice", cache.timeslice, "size", len(tsblocks.Blocks))
					if pos < len(cache.hashes) {
						remain = true
					}

					response.TSBlocks = append(response.TSBlocks, tsblocks)
				} /*  else {
					forwardPack(ack.response.ConfirmPoint.Timeslice)
				} */
			} /* else {
				s.cachelock.RUnlock()
			} */
			//endTimeslice = forwardPack(s, nodeID, ack.response.ConfirmPoint.Timeslice, &response, count)
			//endTimeslice =

			for endTimeslice <= curEndPoint && count < maxSYNCCapLimit {
				endTimeslice++
				tsblocks := &protocol.TimesliceBlocks{}
				tsblocks.TSIndex.Index = 0
				if hashes, err := s.blkstorage.GetBlockHashByTmSlice(endTimeslice); err == nil {
					//log.Debug("handleSYNCBlockResponseACK 5")
					tsblocks.TSIndex.Timeslice = endTimeslice
					if len(hashes) == 0 {
						continue
					}

					if len(hashes) <= maxSYNCCapLimit-count {
						/* for _, hash := range hashes {
							log.Trace(">> SYNC-BLOCK-RESPONSE", "nodeID", nodeID, "timeslice", endTimeslice, "hash", hash.String())
						}
						*/
						if blocks, err := s.blkstorage.GetBlocks(hashes); err == nil {
							tsblocks.Blocks = blocks
							tsblocks.TSIndex.Index = uint(len(hashes)) - 1
							response.TSBlocks = append(response.TSBlocks, tsblocks)
							count += len(hashes)
							log.Debug("Handle packing(full)", "nodeID", nodeID, "timeslice", endTimeslice,
								"total", len(hashes), "curPacking", len(blocks), "remain", len(hashes)-len(blocks))
							//log.Debug("Make SYNC-BLOCK-RESPONSE(full)", "nodeID", nodeID, "timeslice", endTimeslice, "size", maxSYNCCapLimit-count)
							continue
						}
					} else {
						s.cachelock.Lock()
						s.sliceCache[nodeID] = &timesliceHash{timeslice: endTimeslice, hashes: hashes}
						s.cachelock.Unlock()

						/* for i := 0; i < maxSYNCCapLimit-count; i++ {

						} */

						if blocks, err := s.blkstorage.GetBlocks(hashes[0 : maxSYNCCapLimit-count]); err == nil {
							tsblocks.Blocks = blocks
							tsblocks.TSIndex.Index = uint(maxSYNCCapLimit - count - 1)
							response.TSBlocks = append(response.TSBlocks, tsblocks)
							count += maxSYNCCapLimit - count

							log.Debug("Handle packing(portion)", "nodeID", nodeID, "timeslice", endTimeslice,
								"total", len(hashes), "curPos", len(blocks), "curPacking", len(blocks), "remain", len(hashes)-len(blocks))

							//log.Debug("Make SYNC-BLOCK-RESPONSE(portion)", "nodeID", nodeID, "timeslice", endTimeslice, "size", maxSYNCCapLimit-count)
						} else {
							log.Debug("Error load block", "nodeID", nodeID, "timeslice", endTimeslice)
						}
						remain = true
						break
					}
				} else {
					log.Debug("Error get local block-hash", "nodeID", nodeID, "timeslice", endTimeslice)
				}
			}
			if endTimeslice >= curEndPoint && !remain {
				response.End = true
				log.Debug("Meet the cur-end-timeslice", "nodeID", nodeID, "endtimeslice", endTimeslice)
			} else {
				response.End = false
			}

			if peer := s.peerset.FindPeer(nodeID); peer != nil {
				if err := peer.SendSYNCBlockResponse(&response); err != nil {
					log.Debug("Error send SYNC-BLOCK-RESPONSE", "nodeID", nodeID, "curTS", ack.response.ConfirmPoint.Timeslice, "err", err)
				} else {
					log.Debug("handleSYNCBlockResponseACK 6")
				}
				// send new syn-blocks-response message
			} else {
				log.Debug("Error find peer", "nodeID", nodeID)
				return fmt.Errorf("peer offline")
			}
		}
	}
	return nil
}

func (s *Synchroniser) Clear(nodeID string) {
	s.cachelock.Lock()
	if _, ok := s.sliceCache[nodeID]; ok {
		delete(s.sliceCache, nodeID)
	}
	s.cachelock.Unlock()
	log.Debug("Clear node", "nodeID", nodeID)
}

func (s *Synchroniser) synchroinise(peer core.Peer, beginTimeslice uint64, endTimeslice uint64, blockNum uint64) {
	s.wg.Add(1)
	defer s.wg.Done()

	atomic.StoreInt32(&s.syncing, 1)
	stat := core.SYNCStatusEvent{
		Progress:          core.SYNC_READY,
		BeginTS:           beginTimeslice,
		EndTS:             endTimeslice,
		CurTS:             0,
		AccumulateSYNCNum: 0,
		BeginTime:         time.Now(),
		CurOrigin:         peer.NodeID() + "[" + peer.Address() + "]",
		//TriedOrigin:       make([]string, 0)
	}

	s.syncEvent.Post(stat)

	var err error
	i := beginTimeslice

	for ; i <= endTimeslice; i++ {
		stat.CurTS = i
		stat.Err = nil
		stat.Progress = core.SYNC_SYNCING
		s.syncEvent.Post(stat)
		//try := 3
		if err = s.syncTimeslice(peer, &stat, i); err != nil {
			stat.Err = err
			stat.EndTime = time.Now()
			stat.Progress = core.SYNC_ERROR
			s.syncEvent.Post(stat)
			break
		}

	}
	if i >= endTimeslice && err == nil {
		stat.Err = nil
		stat.EndTime = time.Now()
		stat.Progress = core.SYNC_END
		s.syncEvent.Post(stat)
	}
	s.done <- struct{}{}
	atomic.StoreInt32(&s.syncing, 0)
}

func (s *Synchroniser) syncTimeslice(p core.Peer, stat *core.SYNCStatusEvent, ts uint64) error {
	var err error
	/* sendloop:
	for {
		select {
		case <-s.cancelCh:
			stat.Err = fmt.Errorf("canceled")
			return stat.Err
		default:
			err = p.RequestBlockHashBySlice(ts)
			if err == nil {
				log.Debug("Error send request-message", "err", err)
				break sendloop
			} else {
				time.Sleep(1 * time.Second)
			}
		}
	} */
	err = p.RequestBlockHashBySlice(ts)
	if err != nil {
		log.Debug("Error send request-message", "err", err)
		return err
	} else {
		log.Debug("Send request-message OK", "timeslice", ts)
	}
	timeout := time.After(s.requestTTL())

	//loop:
	for {

		select {
		case response := <-s.blockhashesCh:
			if all, ok := response.(*TSHashesPacket); ok {
				if all.timeslice != ts {
					continue
				}
				if len(all.hashes) <= 0 {
					return nil
				}
				for _, hash := range all.hashes {
					s.fetcher.MarkAnnounced(hash, p.NodeID())
				}
				var diff []common.Hash
				diff, err = s.blkstorage.GetBlocksDiffSet(ts, all.hashes)
				if err != nil {
					return err
				}
				if len(diff) <= 0 {
					return nil
				}
				//log.Debug("process request 1")
				s.fetcher.MarkFlighting(diff, p.NodeID())
				//log.Debug("process request 2")
				if err = p.RequestBlocksBySlice(all.timeslice, diff); err != nil {
					//log.Debug("process request 3")
					return err
				}

			} else {
				return errInternal
			}
		case response := <-s.blocksCh:
			if blks, ok := response.(*TSBlocksPacket); ok {
				if blks.timeslice != ts {
					continue
				}
				syncResoponseEvent := &core.SYNCResponseEvent{Blocks: make([]types.Block, 0)}
				hashes := make([]common.Hash, 0)
				for _, blk := range blks.blocks {
					//s.mempool.EnQueue(blk)
					if block, err := types.BlockDecode(blk); err == nil {
						syncResoponseEvent.Blocks = append(syncResoponseEvent.Blocks, block)
						hashes = append(hashes, block.GetHash())
					}
				}
				//log.Debug("process response 1")
				s.fetcher.UnMarkFlighting(hashes)
				//log.Debug("process response 2")
				stat.AccumulateSYNCNum = stat.AccumulateSYNCNum + uint64(len(syncResoponseEvent.Blocks))
				if len(syncResoponseEvent.Blocks) > 0 {
					s.blockPoolEvent.Post(syncResoponseEvent)
				}
			}
			log.Debug("timeslice done", "timeslice", ts)
			return nil
		case <-timeout:
			log.Debug("Wait response timeout", "timeslice", ts)
			stat.Err = errSendMsgTimeout
			err = p.RequestBlockHashBySlice(ts)
			if err != nil {
				log.Debug("Error send request-message", "err", err)
				return err
			} else {
				log.Debug("Send request-message OK", "timeslice", ts)
			}
			timeout = time.After(s.requestTTL())
		case <-s.cancelCh:
			stat.Err = fmt.Errorf("canceled")
			return stat.Err
		}
	}

	//return nil
}

func (s *Synchroniser) ExceedAnnounceLimit(node string) bool {
	return s.fetcher.ExceedAnnounceLimit(node)
}

func (s *Synchroniser) RequestBlock(hash common.Hash) error {
	return s.fetcher.AsyncRequestBlock(hash)
}

func (s *Synchroniser) RequestIsolatedBlock(hash common.Hash) error {
	return s.fetcher.AsyncRequestOrphanBlock(hash)
}

func (s *Synchroniser) Broadcast(hash common.Hash) error {
	return s.relayer.broadcast(hash)
}

func (s *Synchroniser) MarkAnnounced(hash common.Hash, nodeID string) {
	s.fetcher.MarkAnnounced(hash, nodeID)
}

func (s *Synchroniser) requestTTL() time.Duration {
	return time.Duration(120 * time.Second)
}

func (s *Synchroniser) DeliverLastTimeSliceResp(id string, timeslice uint64) error {
	log.Trace("Deliver LAST-MAINBLOCK-TIMESLICE response")
	return s.deliverResponse(id, s.peerSliceCh, &TimeslicePacket{peerId: id, timeslice: timeslice})
}

func (s *Synchroniser) DeliverBlockHashesResp(id string, ts uint64, hash []common.Hash) error {
	return s.deliverResponse(id, s.blockhashesCh, &TSHashesPacket{peerId: id, timeslice: ts, hashes: hash})
}

func (s *Synchroniser) DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error {
	return s.deliverResponse(id, s.blocksCh, &TSBlocksPacket{peerId: id, timeslice: ts, blocks: blocks})
}

func (s *Synchroniser) DeliverNewBlockResp(id string, data [][]byte) error {
	return s.deliverResponse(id, s.fetcher.resCh, &NewBlockPacket{peerId: id, blocks: data})
}

func (s *Synchroniser) DeliverSYNCBlockRequest(id string, beginPoint *protocol.TimesliceIndex) error {
	return s.deliverRequest(id, s.syncreqCh, &SYNCBlockReqPacket{peerID: id, beginPoint: beginPoint})
}

func (s *Synchroniser) DeliverSYNCBlockResponse(id string, response *protocol.SYNCBlockResponse) error {
	return s.deliverResponse(id, s.blockresCh, &SYNCBlockRespPacket{peerID: id, response: response})
}

func (s *Synchroniser) DeliverSYNCBlockACKResponse(id string, response *protocol.SYNCBlockResponseACK) error {
	err := s.deliverResponse(id, s.blockresAckCh, &SYNCBlockResACKPacket{peerID: id, response: response})
	log.Debug("deliver ok")
	return err
}

func (s *Synchroniser) DeliverLocatorResponse(id string, response []protocol.MainChainSample) error {
	return s.deliverResponse(id, s.locatorBlockresCh, &LocatorPacket{peerID: id, response: response})
}

// deliverResponse deliver injects a new batch of data received from a remote node.
func (s *Synchroniser) deliverResponse(id string, destCh chan core.Response, response core.Response) (err error) {
	// Update the delivery metrics for both good and failed deliveries
	/* inMeter.Mark(int64(packet.Items()))
	defer func() {
		if err != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}() */
	// Deliver or abort if the sync is canceled while queuing
	s.cancelLock.RLock()
	cancel := s.cancelCh
	s.cancelLock.RUnlock()
	if cancel == nil {
		return errNoSyncActive
	}
	select {
	case destCh <- response:
		log.Trace("Deliver was finished")
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}

// deliverRequest deliver injects a new batch of data received from a remote node.
func (s *Synchroniser) deliverRequest(id string, destCh chan core.Request, request core.Request) (err error) {
	// Update the delivery metrics for both good and failed deliveries
	/* inMeter.Mark(int64(packet.Items()))
	defer func() {
		if err != nil {
			dropMeter.Mark(int64(packet.Items()))
		}
	}() */
	// Deliver or abort if the sync is canceled while queuing
	s.cancelLock.RLock()
	cancel := s.cancelCh
	s.cancelLock.RUnlock()
	if cancel == nil {
		return errNoSyncActive
	}
	select {
	case destCh <- request:
		log.Trace("Deliver was finished")
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}
