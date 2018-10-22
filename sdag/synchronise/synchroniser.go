package synchronise

import (
	"errors"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/event"

	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/mainchain"
)

var (
	errNoSyncActive   = errors.New("no sync active")
	errInternal       = errors.New("internal error")
	errSendMsgTimeout = errors.New("send message timeout")
)

type Synchroniser struct {
	peers      PeerSetI
	mainChain  mainchain.MainChainI
	blkstorage BlockStorageI
	//mempool    BlockPoolI
	blockPoolEvent *event.TypeMux

	oneSliceSyncDone chan uint64

	peerSliceCh   chan RespPacketI
	blockhashesCh chan RespPacketI
	blocksCh      chan RespPacketI
	newBlockCh    chan RespPacketI

	blockReqCh chan struct{}

	blockReqQueue      map[common.Hash]string
	blockUnfinishQueue map[common.Hash]string
	blockQueueLock     sync.RWMutex // Lock to protect the cancel channel and peer in delivers

	cancelCh chan struct{} // Channel to cancel mid-flight syncs

	syncResultCh chan error

	cancelLock sync.RWMutex // Lock to protect the cancel channel and peer in delivers

}

func NewSynchroinser(ps PeerSetI, mc mainchain.MainChainI, bs BlockStorageI, feed *event.TypeMux, resultCh chan error) (*Synchroniser, error) {
	syncer := &Synchroniser{peers: ps,
		mainChain:      mc,
		blkstorage:     bs,
		blockPoolEvent: feed}
	syncer.oneSliceSyncDone = make(chan uint64)
	syncer.peerSliceCh = make(chan RespPacketI)
	syncer.blockhashesCh = make(chan RespPacketI)
	syncer.blocksCh = make(chan RespPacketI)
	syncer.newBlockCh = make(chan RespPacketI, 100)
	syncer.blockReqCh = make(chan struct{})

	syncer.blockReqQueue = make(map[common.Hash]string)
	syncer.blockUnfinishQueue = make(map[common.Hash]string)
	syncer.cancelCh = make(chan struct{})
	syncer.syncResultCh = resultCh
	return syncer, nil
}

func (s *Synchroniser) Start() error {
	go s.loop()
	go s.SyncHeavy()
	return nil
}

func (s *Synchroniser) loop() {
	triedNodes := make(map[string]string)
	//errCh := make(chan error)

	//forloop:
	for {
		select {
		case _ = <-s.blockReqCh:
			// 随机挑选一个节点
			peer, err := s.peers.RandomSelectIdlePeer()
			if err != nil {
				log.Error("Error select idle node", "error", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if _, existed := triedNodes[peer.NodeID()]; existed {
				continue
			} else {
				triedNodes[peer.NodeID()] = peer.NodeID()
			}
			go s.processRequestBlock(peer)
		case packet := <-s.newBlockCh:
			log.Trace("Receive a packet.")
			go s.processBlockResp(packet)
		case <-s.cancelCh:
			return
		}
	}
}

func (s *Synchroniser) processBlockResp(packet RespPacketI) {
	log.Trace("Process block response")
	s.blockQueueLock.Lock()
	defer s.blockQueueLock.Unlock()

	newblockEvent := &core.NewBlocksEvent{Blocks: make([]types.Block, 0)}
	if response, ok := packet.(*NewBlockPacket); ok {
		for _, item := range response.blocks {
			if block, err := types.BlockDecode(item); err == nil {
				newblockEvent.Blocks = append(newblockEvent.Blocks, block)
				delete(s.blockUnfinishQueue, block.GetHash()) //已经在未完成队列中
				log.Trace("Add block to mempool.")
			} else {
				log.Trace("Error Add block to mempool", "err", err)
			}
		}
	}
	if len(newblockEvent.Blocks) > 0 {
		s.blockPoolEvent.Post(newblockEvent)
	}
}

func (s *Synchroniser) processRequestBlock(peer PeerI) {
	defer s.blockQueueLock.Unlock()
	s.blockQueueLock.Lock()
	var req []common.Hash
	for k, v := range s.blockReqQueue {
		delete(s.blockReqQueue, k) //已经在未完成队列中
		if _, ok := s.blockUnfinishQueue[k]; ok {
			continue
		}
		s.blockUnfinishQueue[k] = v
		req = append(req, k)
	}
	if len(req) <= 0 {
		return
	}
	peer.RequestBlocks(req)
}

func (s *Synchroniser) SyncHeavy() error {

	var timesliceEnd uint64
	timeout := time.After(s.requestTTL())
	triedNodes := make(map[string]string)
	errCh := make(chan error)

	lastSyncSlice := s.mainChain.GetLastTempMainBlkSlice() - 32
	if lastSyncSlice < 0 {
		lastSyncSlice = 0
	}

loop:
	for {
		// 随机挑选一个节点
		peer, err := s.peers.RandomSelectIdlePeer()
		if err != nil {
			s.syncResultCh <- err
			return err
		}
		if _, existed := triedNodes[peer.NodeID()]; existed {
			continue
		} else {
			triedNodes[peer.NodeID()] = peer.NodeID()
		}

		// 查询其当前最近一次临时主块的时间片
		go peer.RequestLastMainSlice()

		select {
		case resp := <-s.peerSliceCh:
			if timesliceResp, ok := resp.(*TimeslicePacket); ok {
				if lastSyncSlice > timesliceResp.timeslice {
					continue
				}
				timesliceEnd = utils.GetMainTime(timesliceResp.timeslice)
				go s.syncTimeslice(peer, lastSyncSlice, errCh)
			}
		case _ = <-errCh:
			continue
		case ts := <-s.oneSliceSyncDone:
			if ts >= timesliceEnd {
				break loop
			}
			lastSyncSlice = ts + 1
			go s.syncTimeslice(peer, lastSyncSlice, errCh)
		case <-timeout:
			continue
		case <-s.cancelCh:
			return nil
		}
	}
	s.syncResultCh <- nil
	return nil
}

func (s *Synchroniser) syncTimeslice(p PeerI, ts uint64, errCh chan error) {
	err := p.RequestBlockHashBySlice(ts)
	if err != nil {
		errCh <- err
		return
	}
	timeout := time.After(s.requestTTL())

	select {
	case response := <-s.blockhashesCh:
		if all, ok := response.(*SliceBlkHashesPacket); ok {
			var diff []common.Hash
			diff, err = s.blkstorage.GetBlocksDiffSet(ts, all.hashes)
			if err = p.RequestBlocksBySlice(all.timeslice, diff); err != nil {
				errCh <- err
				return
			}
		} else {
			errCh <- errInternal
			return
		}
	case response := <-s.blocksCh:
		if blks, ok := response.(*SliceBlkDatasPacket); ok {
			newblockEvent := &core.NewBlocksEvent{Blocks: make([]types.Block, 0)}
			for _, blk := range blks.blocks {
				//s.mempool.EnQueue(blk)
				if block, err := types.BlockDecode(blk); err == nil {
					newblockEvent.Blocks = append(newblockEvent.Blocks, block)
				}
			}
			if len(newblockEvent.Blocks) > 0 {
				s.blockPoolEvent.Post(newblockEvent)
			}
		}
	case <-timeout:
		errCh <- errSendMsgTimeout
		return
	}
	s.oneSliceSyncDone <- ts
}

func (s *Synchroniser) AsyncRequestBlock(hash common.Hash) error {
	s.blockQueueLock.Lock()
	s.blockReqQueue[hash] = ""
	s.blockQueueLock.Unlock()

	s.blockReqCh <- struct{}{}
	return nil
}

func (s *Synchroniser) requestTTL() time.Duration {
	return time.Duration(2000)
}

func (s *Synchroniser) DeliverLastTimeSliceResp(id string, timeslice uint64) error {
	return s.deliverResponse(id, s.peerSliceCh, &TimeslicePacket{peerId: id, timeslice: timeslice})
}

func (s *Synchroniser) DeliverBlockHashesResp(id string, ts uint64, hash []common.Hash) error {
	return s.deliverResponse(id, s.blockhashesCh, &SliceBlkHashesPacket{peerId: id, timeslice: ts, hashes: hash})
}

func (s *Synchroniser) DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error {
	return s.deliverResponse(id, s.blocksCh, &SliceBlkDatasPacket{peerId: id, timeslice: ts, blocks: blocks})
}

func (s *Synchroniser) DeliverNewBlockResp(id string, data [][]byte) error {
	return s.deliverResponse(id, s.newBlockCh, &NewBlockPacket{peerId: id, blocks: data})
}

// deliver injects a new batch of data received from a remote node.
func (s *Synchroniser) deliverResponse(id string, destCh chan RespPacketI, packet RespPacketI) (err error) {
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
	case destCh <- packet:
		log.Trace("Deliver was finished")
		return nil
	case <-cancel:
		return errNoSyncActive
	}

}
