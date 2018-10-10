package sdag

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/synchronise"
	"github.com/TOSIO/go-tos/services/p2p"
)

var (
	errClosed            = errors.New("peer set is closed")
	errAlreadyRegistered = errors.New("peer is already registered")
	errNotRegistered     = errors.New("peer is not registered")
	errNoIdleNode        = errors.New("no idle node")
)

var (
	handshakeTimeout = 5 * time.Second
)

// peer实现节点对节点的业务逻辑（交易转发、请求区块、握手等）
type peer struct {
	id string

	*p2p.Peer
	rw p2p.MsgReadWriter

	version  int         // Protocol version negotiated
	forkDrop *time.Timer // Timed connection dropper if forks aren't validated in time

	blocksQueue chan []byte

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	term chan struct{} // Termination channel to stop the broadcaster
}

// peerSet represents the collection of active peers currently participating in
// the tos sub-protocol.
type peerSet struct {
	peers  map[string]*peer
	lock   sync.RWMutex
	closed bool
}

func newPeer(version int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		Peer:        p,
		rw:          rw,
		version:     version,
		id:          fmt.Sprintf("%x", p.ID().Bytes()[:8]),
		term:        make(chan struct{}),
		blocksQueue: make(chan []byte, 100),
	}
}

// close signals the broadcast goroutine to terminate.
func (p *peer) close() {
	close(p.term)
}

// 节点传输的业务逻辑在此实现
func (p *peer) broadcast() {
	p.Log().Info("Peer.broadcast() | called.")
	for {
		select {
		case block := <-p.blocksQueue:
			var blocks [][]byte
			blocks = append(blocks, block)
			err := p.SendNewBlocks(blocks)
			p.Log().Trace("Peer.broadcast() | send new block", "nodeID", p.id, "err", err)
		case <-p.term:
			p.Log().Info("Peer.broadcast() exit.")
			return
		}
	}
}

func (p *peer) Handshake(network uint64) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	data := "hello"

	go func() {
		errc <- p2p.Send(p.rw, StatusMsg, &data) //p.rw == protoRW
	}()
	go func() {
		msg, err := p.rw.ReadMsg()
		p.Log().Info("recv message : ", msg)
		errc <- err
	}()
	timeout := time.NewTimer(handshakeTimeout)
	defer timeout.Stop()
	for i := 0; i < 2; i++ {
		select {
		case err := <-errc:
			if err != nil {
				return err
			}
		case <-timeout.C:
			return p2p.DiscReadTimeout
		}
	}
	return nil
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

func (ps *peerSet) RandomSelectIdlePeer() (synchronise.PeerI, error) {
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

func (ps *peerSet) Peers() map[string]*peer {
	ps.lock.Lock()
	defer ps.lock.Unlock()
	ret := make(map[string]*peer)
	for k, v := range ps.peers {
		ret[k] = v
	}
	return ret
}

// 发送时间片点
func (p *peer) SendTimeSlice(slice uint64) error {
	return p2p.Send(p.rw, LastMainTimeSlice, slice)
}

func (p *peer) SendNewBlocks(blocks [][]byte) error {
	return p2p.Send(p.rw, NewBlockMsg, blocks)
}

func (p *peer) SendBlockHashes(timeslice uint64, hashes []common.Hash) error {
	return p2p.Send(p.rw, BlockHashBySliceMsg, &GetBlockHashBySliceResp{Timeslice: timeslice, Hashes: hashes})
}

func (p *peer) SendSliceBlocks(timeslice uint64, blocks [][]byte) error {
	return p2p.Send(p.rw, GetBlocksBySliceMsg, &GetBlockDataBySliceResp{Timeslice: timeslice, Blocks: blocks})
}

func (p *peer) RequestBlocksBySlice(timeslice uint64, hashes []common.Hash) error {
	return p2p.Send(p.rw, GetBlocksBySliceMsg, &GetBlockDataBySliceReq{Timeslice: timeslice, Hashes: hashes})
}

func (p *peer) RequestBlocks(hashes []common.Hash) error {
	return p2p.Send(p.rw, GetBlockByHashMsg, hashes)
}

func (p *peer) NodeID() string {
	return p.id
}

func (p *peer) SetIdle(idle bool) {

}

func (p *peer) RequestBlockHashBySlice(slice uint64) error {
	return p2p.Send(p.rw, GetBlockHashBySliceMsg, slice)
}

func (p *peer) RequestLastMainSlice() error {
	return p2p.Send(p.rw, GetLastMainTimeSlice, nil)
}

func (p *peer) AsyncSendBlock(block []byte) {
	select {
	case p.blocksQueue <- block:
		log.Trace("func peer.AsyncSendBlock | send block to channel success", "nodeID", p.id)
	default:
		log.Debug("peer [%s]'s queue is full, so give up.", p.id)
	}
}
