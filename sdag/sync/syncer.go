package sync

import (
	"time"

	"github.com/TOSIO/go-tos/sdag/mainchain"
)

type Syncer struct {
	peers     PeerSetI
	mainChain mainchain.MainChainI

	peerSliceCh chan int64
}

func (s *Syncer) syncHeavy() error {
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
		case peerTmSlice := <-s.peerSliceCh:
			if s.mainChain.GetLastTempMainBlkSlice() > peerTmSlice {
				return nil
			}
			break loop
		case <-timeout:
			return nil
		}
	}

	return nil
}

func (d *Syncer) requestTTL() time.Duration {
	return time.Duration(2000)
}
