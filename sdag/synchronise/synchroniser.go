package synchronise

import (
	"errors"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/utils"
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
	mempool    MemPoolI

	oneSliceSyncDone chan uint64

	peerSliceCh   chan RespPacketI
	blockhashesCh chan RespPacketI
	blocksCh      chan RespPacketI
	newBlockCh    chan RespPacketI

	cancelCh   chan struct{} // Channel to cancel mid-flight syncs
	cancelLock sync.RWMutex  // Lock to protect the cancel channel and peer in delivers

}

func NewSynchroinser(ps PeerSetI, mc mainchain.MainChainI, bs BlockStorageI, mp MemPoolI) (*Synchroniser, error) {
	return &Synchroniser{peers: ps,
		mainChain:  mc,
		blkstorage: bs,
		mempool:    mp}, nil
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
			if err = p.RequestBlockData(all.timeslice, diff); err != nil {
				errCh <- err
				return
			}
		} else {
			errCh <- errInternal
			return
		}
	case response := <-s.blocksCh:
		if blks, ok := response.(*SliceBlkDatasPacket); ok {
			for _, blk := range blks.blocks {
				s.mempool.AddBlock(blk)
			}
		}
	case newBlock := <-s.newBlockCh:
		if blk, ok := newBlock.(*NewBlockPacket); ok {
			s.mempool.AddBlock(blk.block)
		}
	case <-timeout:
		errCh <- errSendMsgTimeout
		return
	}
	s.oneSliceSyncDone <- ts
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

func (s *Synchroniser) DeliverNewBlockResp(id string, data []byte) error {
	return s.deliverResponse(id, s.newBlockCh, &NewBlockPacket{peerId: id, block: data})
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
		return nil
	case <-cancel:
		return errNoSyncActive
	}
}
