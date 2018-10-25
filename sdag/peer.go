package sdag

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"

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

	blocksQueue chan []byte // relay

	//knownBlocks mapset.Set

	head common.Hash
	td   *big.Int
	lock sync.RWMutex

	term chan struct{} // Termination channel to stop the broadcaster
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
	p.Log().Info("Starting broadcast")
	for {
		select {
		case block := <-p.blocksQueue:
			var blocks [][]byte
			blocks = append(blocks, block)
			err := p.SendNewBlocks(blocks)
			p.Log().Trace("Peer.broadcast() | send new block", "nodeID", p.id, "err", err)
		case <-p.term:
			p.Log().Info("Broadcast stopped")
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

// 发送时间片点
func (p *peer) SendTimeSlice(slice uint64) error {
	return p2p.Send(p.rw, LastMainTimeSlice, slice)
}

func (p *peer) SendNewBlocks(blocks [][]byte) error {
	return p2p.Send(p.rw, NewBlockMsg, blocks)
}

func (p *peer) SendNewBlockHash(hash common.Hash) error {
	return p2p.Send(p.rw, NewBlockHashMsg, hash)
}

func (p *peer) SendBlockHashes(timeslice uint64, hashes []common.Hash) error {
	return p2p.Send(p.rw, BlockHashBySliceMsg, &GetBlockHashBySliceResp{Timeslice: timeslice, Hashes: hashes})
}

func (p *peer) SendSliceBlocks(timeslice uint64, blocks [][]byte) error {
	return p2p.Send(p.rw, BlocksBySliceMsg, &GetBlockDataBySliceResp{Timeslice: timeslice, Blocks: blocks})
}

func (p *peer) RequestBlocksBySlice(timeslice uint64, hashes []common.Hash) error {
	return p2p.Send(p.rw, GetBlocksBySliceMsg, &GetBlockDataBySliceReq{Timeslice: timeslice, Hashes: hashes})
}

/* func (p *peer) RequestBlocks(hashes []common.Hash) error {
	return p2p.Send(p.rw, GetBlockByHashMsg, hashes)
} */

func (p *peer) RequestBlock(hash common.Hash) error {
	hashes := make([]common.Hash, 0)
	hashes = append(hashes, hash)
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

	err := p2p.Send(p.rw, GetLastMainTimeSlice, GetLastMainBlockTSReq{})
	p.Log().Debug(">> GET-LAST-MAINBLOCK-TIMESLICE", "err", err)
	return err
}

func (p *peer) AsyncSendBlock(block []byte) {
	select {
	case p.blocksQueue <- block:
		p.Log().Trace("Send block to channel success")
	default:
		//log.Debug("peer [%s]'s queue is full, so give up.", p.id)
	}
}

/* func (p *peer) IsKnown(hash common.Hash) bool {
	return p.knownBlocks.Contains(hash)
}
*/
