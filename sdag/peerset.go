package sdag

import (
	"math/rand"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/services/p2p"
)

// peerSet represents the collection of active peers currently participating in
// the tos sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

// newPeerSet creates a new peer set to track the active participants.
func newPeerSet() *peerSet {
	return &peerSet{
		peers: make(map[string]*peer),
	}
}

// Register injects a new peer into the working set, or returns an error if the
// peer is already known. If a new peer it registered, its broadcast loop is also
// started.
func (ps *peerSet) Register(p *peer) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	if ps.closed {
		return errClosed
	}
	if _, ok := ps.peers[p.id]; ok {
		return errAlreadyRegistered
	}
	ps.peers[p.id] = p
	go p.broadcast()

	return nil
}

// Unregister removes a remote peer from the active set, disabling any further
// actions to/from that particular entity.
func (ps *peerSet) Unregister(id string) error {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	p, ok := ps.peers[id]
	if !ok {
		return errNotRegistered
	}
	delete(ps.peers, id)
	p.close()

	return nil
}

// Peer retrieves the registered peer with the given id.
func (ps *peerSet) Peer(id string) *peer {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return ps.peers[id]
}

// Len returns if the current number of peers in the set.
func (ps *peerSet) Len() int {
	ps.lock.RLock()
	defer ps.lock.RUnlock()

	return len(ps.peers)
}

func (ps *peerSet) Close() {
	ps.lock.Lock()
	defer ps.lock.Unlock()

	for _, p := range ps.peers {
		p.Disconnect(p2p.DiscQuitting)
	}
	ps.closed = true
}

func (ps *peerSet) RandomSelectIdlePeer() (core.Peer, error) {
	//var ret synchronise.PeerI = ps.Peer("")
	var peers []*peer
	for _, peer := range ps.peers {
		peers = append(peers, peer)
	}

	if len(peers) <= 0 {
		return nil, errNoIdleNode
	}

	rand.Seed(time.Now().UnixNano())
	index := rand.Intn(len(peers))
	return peers[index], nil
}

func (ps *peerSet) FindPeer(id string) core.Peer {
	peer := ps.Peer(id)
	if peer != nil {
		return peer
	}
	return nil
}

func (ps *peerSet) Peers() map[string]core.Peer {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ret := make(map[string]core.Peer)
	for k, v := range ps.peers {
		ret[k] = v
	}
	return ret
}
