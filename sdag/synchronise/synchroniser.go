package synchronise

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/utils"

	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
)

var (
	errNoSyncActive   = errors.New("no sync active")
	errInternal       = errors.New("internal error")
	errSendMsgTimeout = errors.New("send message timeout")
)

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

	//syncResultCh chan error
	genesisTimeslice uint64
	genesis          common.Hash

	cancelLock sync.RWMutex   // Lock to protect the cancel channel and peer in delivers
	wg         sync.WaitGroup // for shutdown sync
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

func (s *Synchroniser) schedule(tasks map[string]*core.NewSYNCTask) {
	nowTimeslice := utils.GetMainTime(utils.GetTOSTimeStamp())
	lastTempMBTimeslice := s.mainChain.GetLastTempMainBlkSlice()
	if lastTempMBTimeslice >= nowTimeslice {
		s.netFeed.Send(core.SDAGSYNC_COMPLETED)
		log.Info("Synchronise is completed", "lastTS", s.mainChain.GetLastTempMainBlkSlice(), "nowTS", nowTimeslice)
		return
	}

	if atomic.LoadInt32(&s.syncing) != 0 {
		log.Debug("Synchroniser is in synchronizing")
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

	if target != "" && origin.LastMainBlockNum > s.mainChain.GetMainTail().Number &&
		origin.LastTempMBTimeslice > s.mainChain.GetLastTempMainBlkSlice()+3 {
		beginTimeslice := s.mainChain.GetLastTempMainBlkSlice() - 32
		if beginTimeslice < 0 || beginTimeslice <= s.genesisTimeslice {
			beginTimeslice = origin.FirstMBTimeslice
			log.Debug("Adjust the begin timeslice", "begin", beginTimeslice)
		}
		s.netFeed.Send(core.SDAGSYNC_SYNCING)
		go s.synchroinise(peer, beginTimeslice, origin.LastTempMBTimeslice, origin.LastMainBlockNum)
	} else {
		for key, _ := range tasks {
			delete(tasks, key)
		}
	}
}

func (s *Synchroniser) loop() {
	//var lastNetUnavilableTime time.Time
	s.wg.Add(1)
	defer s.wg.Done()

	queuedTask := make(map[string]*core.NewSYNCTask)

	for {
		s.schedule(queuedTask)
		select {
		case newTask := <-s.newTaskSub.Chan():
			if task, ok := newTask.Data.(*core.NewSYNCTask); ok {
				queuedTask[task.NodeID] = task
			}
		case <-s.done:
			continue
		case <-s.quitCh:
			return
		}
	}
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
				newblockEvent := &core.NewBlocksEvent{Blocks: make([]types.Block, 0)}
				hashes := make([]common.Hash, 0)
				for _, blk := range blks.blocks {
					//s.mempool.EnQueue(blk)
					if block, err := types.BlockDecode(blk); err == nil {
						newblockEvent.Blocks = append(newblockEvent.Blocks, block)
						hashes = append(hashes, block.GetHash())
					}
				}
				//log.Debug("process response 1")
				s.fetcher.UnMarkFlighting(hashes)
				//log.Debug("process response 2")
				stat.AccumulateSYNCNum = stat.AccumulateSYNCNum + uint64(len(newblockEvent.Blocks))
				if len(newblockEvent.Blocks) > 0 {
					s.blockPoolEvent.Post(newblockEvent)
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

func (s *Synchroniser) RequestBlock(hash common.Hash) error {
	return s.fetcher.AsyncRequestBlock(hash)
}

func (s *Synchroniser) Broadcast(hash common.Hash) error {
	return s.relayer.broadcast(hash)
}

func (s *Synchroniser) MarkAnnounced(hash common.Hash, nodeID string) {
	s.fetcher.MarkAnnounced(hash, nodeID)
}

func (s *Synchroniser) requestTTL() time.Duration {
	return time.Duration(10 * time.Second)
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

// deliver injects a new batch of data received from a remote node.
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
