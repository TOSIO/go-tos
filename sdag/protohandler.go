package sdag

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/TOSIO/go-tos/sdag/mainchain"
	"github.com/TOSIO/go-tos/sdag/synchronise"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/services/p2p"
	"github.com/TOSIO/go-tos/services/p2p/discover"
)

var errIncompatibleConfig = errors.New("incompatible configuration")

// tos sdag协议管理、实现
type ProtocolManager struct {
	networkID uint64

	maxPeers int

	peers *peerSet

	synchroniser synchronise.SynchroniserI
	blkstorage   synchronise.BlockStorageI
	SubProtocols []p2p.Protocol

	mainChain mainchain.MainChainI
	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg sync.WaitGroup
}

// NodeInfo represents a short summary of the tos sub-protocol metadata
// known about the host peer.
type NodeInfo struct {
	Network    uint64      `json:"network"`    // tos network ID (1=Frontier, 2=Morden, Ropsten=3, Rinkeby=4)
	Difficulty *big.Int    `json:"difficulty"` // Total difficulty of the host's blockchain
	Genesis    common.Hash `json:"genesis"`    // SHA3 hash of the host's genesis block
	//Config     *params.ChainConfig `json:"config"`     // Chain configuration for the fork rules
	//Head       common.Hash         `json:"head"`       // SHA3 hash of the host's best owned block
}

// NewProtocolManager returns a new tos sub protocol manager. The tos sub protocol manages peers capable
// with the tos network.
func NewProtocolManager(config *interface{}, networkID uint64) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkID:   networkID,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		quitSync:    make(chan struct{}),
	}

	// Initiate a sub-protocol for every implemented version we can handle
	if err := manager.initProtocols(); err != nil {
		return nil, err
	}
	return manager, nil
}

// this function will be removed in future
func (pm *ProtocolManager) consumeNewPeer() {
	for {
		select {
		case <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			log.Trace("func ProtocolManager.consumeNewPeer | called.")
		case <-pm.noMorePeers:
			return
		}
	}
}

func (pm *ProtocolManager) initProtocols() error {
	pm.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {

		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		pm.SubProtocols = append(pm.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := pm.newPeer(int(version), p, rw)
				select {
				case pm.newPeerCh <- peer:
					pm.wg.Add(1)
					defer pm.wg.Done()
					return pm.handle(peer)
				case <-pm.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return pm.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := pm.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(pm.SubProtocols) == 0 {
		return errIncompatibleConfig
	}
	return nil
}

func (pm *ProtocolManager) removePeer(id string) {
	log.Info("ProtocolManager.removePeer called.")
	// Short circuit if the peer was already removed
	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Info("Removing TOS peer", "peer", id)

	// Unregister the peer from the downloader and TOS peer set
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}
	// Hard disconnect at the networking layer
	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	log.Info("ProtocolManager.Start called.")
}

func (pm *ProtocolManager) Stop() {
	log.Info("ProtocolManager.Stop called.")

	log.Info("Stopping TOS protocol")

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	// Quit fetcher, txsyncLoop.
	close(pm.quitSync)

	// Disconnect existing sessions.
	// This also closes the gate for any new registrations on the peer set.
	// sessions which are already established but not added to pm.peers yet
	// will exit when they try to register.
	pm.peers.Close()

	// Wait for all peer handler goroutines and the loops to come down.
	pm.wg.Wait()

	log.Info("TOS protocol stopped")

}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	p.Log().Info("ProtocolManager.newPeer called.")
	return newPeer(pv, p, rw)
}

func (pm *ProtocolManager) Peers() *peerSet {
	return pm.peers
}

func (pm *ProtocolManager) handle(p *peer) error {
	p.Log().Info("ProtocolManager.handle called.")
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("TOS peer connected", "name", p.Name())

	// Execute the TOS handshake

	if err := p.Handshake(pm.networkID); err != nil {
		p.Log().Debug("TOS handshake failed", "err", err)
		return err
	}
	/* if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	} */
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("TOS peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("TOS message handling failed", "err", err)
			return err
		}
	}
}
func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

func (pm *ProtocolManager) handleMsg(p *peer) error {
	p.Log().Info("ProtocolManager.handleMsg called.")
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()
	p.Log().Info("ProtocolManager.handleMsg() | receive message,code : ", msg.Code)
	//dispatch message here
	switch msg.Code {
	case GetLastMainTimeSlice: //获取最近一次临时主块的时间片
		return pm.handleGetLastMainTimeSlice(p, msg)
	case GetBlockHashBySliceMsg: //获取时间片对应的所有区块hash
		return pm.handleGetBlockHashBySlice(p, msg)
	case GetBlockDataBySliceMsg: //获取区块数据
		return pm.handleGetBlockDataByHash(p, msg)
	case BlockDataBySliceMsg:
		return pm.handleBlockDatasByHash(p, msg)
	}
	return nil
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	return &NodeInfo{
		Network:    pm.networkID,
		Difficulty: nil,
	}
}

// 获取最近一次临时主块所在时间片消息处理
func (pm *ProtocolManager) handleGetLastMainTimeSlice(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleGetLastMainTimeSlice | Process the last main timeslice query.")
	lastMainSlice := pm.mainChain.GetLastTempMainBlkSlice()
	return p.SendTimeSlice(lastMainSlice)
}

// 获取最近一次临时主块所在时间片消息处理
func (pm *ProtocolManager) handleLastMainTimeSlice(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleLastMainTimeSlice | Process the last main timeslice response.")
	var peerTimeSlice uint64
	err := msg.Decode(&peerTimeSlice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	// 将回复结果递送到同步器
	return pm.synchroniser.DeliverLastTimeSliceResp(p.id, peerTimeSlice)
}

// 根据时间片获取对应所有区块hash消息处理
func (pm *ProtocolManager) handleGetBlockHashBySlice(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleGetBlockHashBySlice | Process block hash query.")
	//lastMainSlicer := pm.mainChain.GetLastTempMainBlkSlice()
	var targetSlice uint64 = 0
	err := msg.Decode(&targetSlice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	var hashes []common.Hash
	hashes, err = pm.blkstorage.GetBlockHashByTmSlice(targetSlice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	return p.SendBlockHashes(targetSlice, hashes)
}

// 根据时间片获取对应所有区块hash消息处理
func (pm *ProtocolManager) handleBlockHashBySlice(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleBlockHashBySlice | Process block hashes response.")
	var response GetBlockHashBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	// 将回复结果递送到同步器
	return pm.synchroniser.DeliverBlockHashesResp(p.id, response.Timeslice, response.Hashes)
}

// 根据区块hash返回对应区块（字节流）消息处理
func (pm *ProtocolManager) handleGetBlockDataByHash(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleGetBlockDataByHash | Process block data query.")
	var req GetBlockDataBySliceReq
	err := msg.Decode(&req)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	if len(req.Hashes) <= 0 {
		log.Trace("func ProtocolManager.handleGetBlockDataByHash | Param 'Hashes' is empty.")
		return nil
	}
	var blocks [][]byte
	blocks, err = pm.blkstorage.GetBlocks(req.Hashes)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if len(blocks) <= 0 {
		return nil
	}
	// 将结果回复给对方
	return p.SendSliceBlocks(req.Timeslice, blocks)
}

func (pm *ProtocolManager) handleBlockDatasByHash(p *peer, msg p2p.Msg) error {
	log.Trace("func ProtocolManager.handleBlockDatasByHash | Process block hashes response.")
	var response GetBlockDataBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	// 将回复结果递送到同步器
	return pm.synchroniser.DeliverBlockDatasResp(p.id, response.Timeslice, response.Blocks)
}

func (pm *ProtocolManager) RelayBlock(blockRLP []byte) error {
	for _, p := range pm.peers.Peers() {
		p.AsyncSendBlock(blockRLP)
	}
	return nil
}
