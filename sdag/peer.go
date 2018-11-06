package sdag

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/protocol"
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

	//head common.Hash
	//td   *big.Int
	firstMBTimeslice    uint64
	lastTempMBTimeslice uint64
	lastMainBlockNum    uint64
	lastCumulatedDiff   *big.Int

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
			p.Log().Debug(">> NEW-BLOCK", "size", len(blocks), "err", err)
		case <-p.term:
			p.Log().Info("Broadcast stopped")
			return
		}
	}
}

func (p *peer) Handshake(network uint64, genesis common.Hash, firstMBTS uint64, ts uint64, num uint64, diff *big.Int) error {
	// Send out own handshake in a new thread
	errc := make(chan error, 2)
	var status protocol.StatusData

	go func() {
		errc <- p2p.Send(p.rw, protocol.StatusMsg, &protocol.StatusData{
			ProtocolVersion: uint32(p.version),
			NetworkId:       network,
			CurFistMBTS:     firstMBTS,
			CurLastTempMBTS: ts,
			CurMainBlockNum: num,
			CumulateDiff:    diff,
			GenesisBlock:    genesis,
		}) //p.rw == protoRW
	}()
	go func() {
		errc <- p.readStatus(network, &status, genesis)
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
	p.firstMBTimeslice = status.CurFistMBTS
	p.lastTempMBTimeslice = status.CurLastTempMBTS
	p.lastMainBlockNum = status.CurMainBlockNum
	p.lastCumulatedDiff = status.CumulateDiff

	return nil
}

func (p *peer) readStatus(network uint64, status *protocol.StatusData, genesis common.Hash) (err error) {
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code != protocol.StatusMsg {
		return errResp(protocol.ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, protocol.StatusMsg)
	}
	if msg.Size > protocol.ProtocolMaxMsgSize {
		return errResp(protocol.ErrMsgTooLarge, "%v > %v", msg.Size, protocol.ProtocolMaxMsgSize)
	}
	// Decode the handshake and make sure everything matches
	if err := msg.Decode(&status); err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	if status.GenesisBlock != genesis {
		return errResp(protocol.ErrGenesisBlockMismatch, "%x (!= %x)", status.GenesisBlock[:8], genesis[:8])
	}
	if status.NetworkId != network {
		return errResp(protocol.ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, network)
	}
	if int(status.ProtocolVersion) != p.version {
		return errResp(protocol.ErrProtocolVersionMismatch, "%d (!= %d)", status.ProtocolVersion, p.version)
	}
	return nil
}

// 发送时间片点
func (p *peer) SendTimeSlice(slice uint64) error {
	return p2p.Send(p.rw, protocol.LastMainTimeSlice, slice)
}

func (p *peer) SendNewBlocks(blocks [][]byte) error {
	return p2p.Send(p.rw, protocol.NewBlockMsg, blocks)
}

func (p *peer) SendNewBlockHash(hash common.Hash) error {
	return p2p.Send(p.rw, protocol.NewBlockHashMsg, hash)
}

func (p *peer) SendBlockHashes(timeslice uint64, hashes []common.Hash) error {
	return p2p.Send(p.rw, protocol.BlockHashBySliceMsg, &protocol.GetBlockHashBySliceResp{Timeslice: timeslice, Hashes: hashes})
}

func (p *peer) SendSliceBlocks(timeslice uint64, blocks [][]byte) error {
	return p2p.Send(p.rw, protocol.BlocksBySliceMsg, &protocol.GetBlockDataBySliceResp{Timeslice: timeslice, Blocks: blocks})
}

func (p *peer) RequestBlocksBySlice(timeslice uint64, hashes []common.Hash) error {
	err := p2p.Send(p.rw, protocol.GetBlocksBySliceMsg, &protocol.GetBlockDataBySliceReq{Timeslice: timeslice, Hashes: hashes})
	p.Log().Debug(">> GET-BLOCK-BY-TIMESLICE-HASH", "timeslice", timeslice, "hash.size", len(hashes), "err", err)
	return err
}

func (p *peer) RequestBlock(hash common.Hash) error {
	hashes := make([]common.Hash, 0)
	hashes = append(hashes, hash)
	p.Log().Debug(">> GET-BLOCK-BY-HASH", "hash", hash.String())
	return p2p.Send(p.rw, protocol.GetBlockByHashMsg, hashes)
}

func (p *peer) NodeID() string {
	return p.id
}
func (p *peer) Address() string {
	return p.RemoteAddr().String()
}

func (p *peer) SetIdle(idle bool) {

}

func (p *peer) RequestBlockHashBySlice(slice uint64) error {
	err := p2p.Send(p.rw, protocol.GetBlockHashBySliceMsg, slice)
	p.Log().Debug(">> GET-BLOCK-HASH-BY-TIMESLICE", "timeslice", slice, "err", err)
	return err
}

func (p *peer) RequestLastMainSlice() error {
	err := p2p.Send(p.rw, protocol.GetLastMainTimeSlice, protocol.GetLastMainBlockTSReq{})
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

func (p *peer) SendSYNCBlockRequest(timeslice uint64, index uint) error {
	err := p2p.Send(p.rw, protocol.SYNCBlockRequestMsg, protocol.SYNCBlockRequest{BeginPoint: protocol.TimesliceIndex{Timeslice: timeslice, Index: index}})
	p.Log().Debug(">> SYNC-BLOCK-REQUEST", "err", err)
	return err
}

func (p *peer) SendSYNCBlockResponse(packet *protocol.SYNCBlockResponse) error {
	err := p2p.Send(p.rw, protocol.SYNCBlockResponseMsg, packet)
	p.Log().Debug(">> SYNC-BLOCK-RESPONSE", "size", len(packet.TSBlocks), "err", err)
	return err
}

func (p *peer) SendSYNCBlockResponseACK(timeslice uint64, index uint) error {
	err := p2p.Send(p.rw, protocol.SYNCBlockResponseACKMsg, protocol.SYNCBlockResponseACK{ConfirmPoint: protocol.TimesliceIndex{Timeslice: timeslice, Index: index}})
	p.Log().Debug(">> SYNC-BLOCK-RESPONSE-ACK", "err", err)
	return err
}
