package synchronise

import (
	"sync"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core"
)

type Relayer struct {
	relayCh chan common.Hash

	fetcher *Fetcher
	peerset core.PeerSet

	quit chan struct{}

	wg sync.WaitGroup
}

func NewRelayer(f *Fetcher, peers core.PeerSet) *Relayer {
	relayer := &Relayer{
		fetcher: f,
		peerset: peers,
		relayCh: make(chan common.Hash, 8),
	}
	return relayer
}

func (r *Relayer) stop() {
	r.quit <- struct{}{}
	r.wg.Wait()
}

func (r *Relayer) loop() {
	r.wg.Add(1)
	defer r.wg.Done()
	for {
		select {
		case hash := <-r.relayCh:
			go r.relay(hash)
		case <-r.quit:
			return
		}
	}
}

func (r *Relayer) findExceptTarget(hash common.Hash) []core.Peer {
	result := make([]core.Peer, 0)

	all := r.peerset.Peers()
	except := r.fetcher.whoAnnounced(hash)
	for id, peer := range all {
		if except == nil || !except.Contains(id) {
			result = append(result, peer)
		}
	}
	return result
}

func (r *Relayer) relay(hash common.Hash) error {
	peers := r.findExceptTarget(hash)
	for _, peer := range peers {
		err := peer.SendNewBlockHash(hash)
		log.Debug(">> Broadcast block", "hash", hash.String(), "targetNode", peer.NodeID(), "err", err)
	}
	return nil
}

func (r *Relayer) broadcast(hash common.Hash) error {
	r.relayCh <- hash
	return nil
}
