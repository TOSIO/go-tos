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
	errNoSyncActive = errors.New("no sync active")
)

type Synchroniser struct {
	peers      PeerSetI
	mainChain  mainchain.MainChainI
	blkstorage BlockStorageI

	peerSliceCh   chan RespPacketI
	blockhashesCh chan RespPacketI
	blocksCh      chan RespPacketI
	cancelCh      chan struct{} // Channel to cancel mid-flight syncs
	cancelLock    sync.RWMutex  // Lock to protect the cancel channel and peer in delivers

}

func (s *Synchroniser) SyncHeavy() error {
	// 随机挑选一个节点
	peer, err := s.peers.SelectRandomPeer()
	if err != nil {
		return err
	}
	// 查询其当前最近一次临时主块的时间片
	go peer.RequestLastMainSlice()

	timeout := time.After(s.requestTTL())
loop:
	for {
		select {
		case resp := <-s.peerSliceCh:
			if timeSliceResp, ok := resp.(*TimeSlicePacket); ok {
				lastMainSlice := s.mainChain.GetLastTempMainBlkSlice()
				if lastMainSlice > timeSliceResp.timeSlice {
					return nil
				}
				syncEndPoint := utils.GetMainTime(timeSliceResp.timeSlice)
				for i := lastMainSlice - 32; i < syncEndPoint; i++ {

				}
			}

			/* if s.mainChain.GetLastTempMainBlkSlice() > peerTmSlice {
				return nil
			} */
			break loop
		case <-timeout:
			return nil
		}
	}

	return nil
}

func (s *Synchroniser) syncByTimeslice(p PeerI, ts uint64) error {
	err := p.RequestBlockHashBySlice(ts)
	if err != nil {
		return nil
	}
	for {
		select {
		case response := <-s.blockhashesCh:
			if all, ok := response.(*SliceBlkHashesPacket); ok {
				var diff []common.Hash
				diff, err = s.blkstorage.GetBlocksDiffSet(ts, all.hashes)
				p.RequestBlockData(all.timeslice, diff)
			}
		case response := <-s.blocksCh:
			if blks, ok := response.(*SliceBlkDatasPacket); ok {
				for _, blk := range blks.blocks {
					s.blkstorage.AddBlock(blk)
				}
			}
		}

	}
	return nil
}

func (s *Synchroniser) requestTTL() time.Duration {
	return time.Duration(2000)
}

func (s *Synchroniser) DeliverLastTimeSliceResp(id string, timeslice uint64) error {
	return s.deliverResponse(id, s.peerSliceCh, &TimeSlicePacket{peerId: id, timeSlice: timeslice})
}

func (s *Synchroniser) DeliverBlockHashesResp(id string, ts uint64, hash []common.Hash) error {
	return s.deliverResponse(id, s.blockhashesCh, &SliceBlkHashesPacket{peerId: id, timeslice: ts, hashes: hash})
}

func (s *Synchroniser) DeliverBlockDatasResp(id string, ts uint64, blocks [][]byte) error {
	return s.deliverResponse(id, s.blockhashesCh, &SliceBlkDatasPacket{peerId: id, timeslice: ts, blocks: blocks})
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
