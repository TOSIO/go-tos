package sdag

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/TOSIO/go-tos/sdag/core"

	"github.com/TOSIO/go-tos/devbase/event"

	"github.com/TOSIO/go-tos/sdag/mainchain"
	"github.com/TOSIO/go-tos/sdag/synchronise"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/services/p2p"
	"github.com/TOSIO/go-tos/services/p2p/discover"
)

var errIncompatibleConfig = errors.New("incompatible configuration")

const (
	STAT_NONE = iota
	STAT_SYNCING
	STAT_READY
	STAT_WORKING
	STAT_NET_UNVAILABLE
)

type status struct {
	nodeNum  int `json:node_num`
	progress int `json:progress` //0-none,1-syncing,2-ready,3-working,4-connecting
}

// tos sdag协议管理、实现
type ProtocolManager struct {
	networkID uint64

	maxPeers int

	peers *peerSet

	blockPoolEvent *event.TypeMux
	relaySub       *event.TypeMuxSubscription
	getSub         *event.TypeMuxSubscription

	syncEvent   *event.TypeMux
	syncstatSub *event.TypeMuxSubscription

	networkFeed *event.Feed
	feeded      bool

	chainDb tosdb.Database // Block chain database

	blockChain   mainchain.MainChainI
	synchroniser synchronise.SynchroniserI
	blkstorage   synchronise.BlockStorageI
	SubProtocols []p2p.Protocol

	mainChain mainchain.MainChainI
	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *peer
	quitSync    chan struct{}
	noMorePeers chan struct{}

	stat       status
	syncResult chan error
	// wait group is used for graceful shutdowns during downloading
	// and processing
	wg sync.WaitGroup
}

func (pm *ProtocolManager) RealNodeIdMessage() []string {
	var peerId []string
	peerSetMessage := pm.peers
	peerMessage := peerSetMessage.peers
	for _, peerIdMessage := range peerMessage {
		peerId = append(peerId, peerIdMessage.id)
	}
	return peerId

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
func NewProtocolManager(config *interface{}, networkID uint64, chain mainchain.MainChainI,
	db tosdb.Database, feed *event.Feed, poolFeed *event.TypeMux) (*ProtocolManager, error) {
	// Create the protocol manager with the base fields
	manager := &ProtocolManager{
		networkID:   networkID,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		quitSync:    make(chan struct{}),

		blockChain:     chain,
		networkFeed:    feed,
		blockPoolEvent: poolFeed,
		feeded:         false,
		stat:           status{nodeNum: 0, progress: STAT_NONE},
		syncResult:     make(chan error),
	}

	manager.relaySub = manager.blockPoolEvent.Subscribe(&core.RelayBlocksEvent{})
	manager.getSub = manager.blockPoolEvent.Subscribe(&core.GetBlocksEvent{})

	manager.syncEvent = &event.TypeMux{}
	manager.syncstatSub = manager.syncEvent.Subscribe(core.SYNCStatusEvent{})
	// Initiate a sub-protocol for every implemented version we can handle
	var err error
	if err = manager.initProtocols(); err != nil {
		return nil, err
	}
	var storageProxy *synchronise.StorageProxy
	//var mempoolProxy *synchronise.MemPoolProxy
	if storageProxy, err = synchronise.NewStorage(db); err != nil {
		log.Error("Initialising Sdag storageproxy failed.")
		return nil, err
	}
	/* if mempoolProxy, err = synchronise.NewMempol(); err != nil {
		log.Error("Initialising Sdag mempool failed.")
		return nil, err
	} */

	if manager.synchroniser, err = synchronise.NewSynchroinser(manager.Peers(),
		chain, storageProxy, feed, poolFeed, manager.syncEvent); err != nil {
		log.Error("Initialising Sdag synchroniser failed.")
		return nil, err
	}

	return manager, nil
}

// this function will be removed in future
func (pm *ProtocolManager) loop() {
	for {
		select {
		case ev := <-pm.relaySub.Chan():
			if event, ok := ev.Data.(*core.RelayBlocksEvent); ok {
				for _, block := range event.Blocks {
					pm.synchroniser.Broadcast(block.GetHash())
				}
			}
		case ev := <-pm.getSub.Chan():
			if event, ok := ev.Data.(*core.GetBlocksEvent); ok {
				for _, block := range event.Hashes {
					//log.Debug("Request", "peer.id", peer.NodeID)
					pm.synchroniser.RequestBlock(block)
				}
			}
		case peer := <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			log.Debug("Receive a new peer,", "peer.id", peer.NodeID())
		case <-pm.noMorePeers:
			return
		case ev := <-pm.syncstatSub.Chan():
			if event, ok := ev.Data.(core.SYNCStatusEvent); ok {
				if event.Progress == core.SYNC_END && event.Err == nil {
					pm.stat.progress = STAT_WORKING
				}
				log.Debug("Synchronizing", "progress", core.SyncCodeToString(event.Progress), "curorigin", event.CurOrigin, "curTS", event.CurTS,
					"startTS", event.BeginTS,
					"endTS", event.EndTS,
					"beginTime", event.BeginTime,
					"endTime", event.EndTime,
					"tiredOrigins", event.TriedOrigin,
					"err", event.Err)

			}
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

	if pm.peers.Len() <= 0 {
		pm.networkFeed.Send(core.NETWORK_CLOSED)
		pm.feeded = false
		pm.stat.progress = STAT_NET_UNVAILABLE
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	log.Info("ProtocolManager.Start called.")
	// start sync procedure
	pm.stat.progress = STAT_SYNCING

	pm.synchroniser.Start()
	go pm.loop()
	pm.maxPeers = maxPeers

}

func (pm *ProtocolManager) Stop() {
	log.Info("ProtocolManager.Stop called.")

	log.Info("Stopping TOS protocol")

	// Quit the sync loop.
	// After this send has completed, no new peers will be accepted.
	pm.noMorePeers <- struct{}{}

	// Quit fetcher, txsyncLoop.
	close(pm.quitSync)

	pm.synchroniser.Stop()

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
	p.Log().Info("Accept new node")
	return newPeer(pv, p, rw)
}

func (pm *ProtocolManager) Peers() *peerSet {
	return pm.peers
}

func (pm *ProtocolManager) handle(p *peer) error {
	p.Log().Info("Handle income-node", "name", p.Name())
	// Ignore maxPeers if this is a trusted peer
	if pm.peers.Len() >= pm.maxPeers && !p.Peer.Info().Network.Trusted {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("TOS peer connected", "name", p.Name())

	// Execute the TOS handshake

	/* if err := p.Handshake(pm.networkID); err != nil {
		p.Log().Debug("TOS handshake failed", "err", err)
		return err
	} */
	/* if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	} */
	// Register the peer locally
	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("TOS peer registration failed", "err", err)
		return err
	}
	if !pm.feeded {
		pm.networkFeed.Send(core.NETWORK_CONNECTED)
		pm.feeded = true
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

func (pm *ProtocolManager) GetStatus() status {
	return pm.stat
}

func (pm *ProtocolManager) handleMsg(p *peer) error {
	//p.Log().Info("Starting handle message")
	msg, err := p.rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	defer msg.Discard()
	p.Log().Debug("Handle message", "msg.Code", MsgCodeToString(int(msg.Code)))
	//dispatch message here
	switch msg.Code {
	case StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(ErrExtraStatusMsg, "uncontrolled status message")
	case GetLastMainTimeSlice: //获取最近一次临时主块的时间片
		return pm.handleGetLastMainTimeSlice(p, msg)
	case GetBlockHashBySliceMsg: //获取时间片对应的所有区块hash
		return pm.handleGetBlockHashBySlice(p, msg)
	case GetBlocksBySliceMsg: //获取区块数据
		return pm.handleGetBlocksBySlice(p, msg)
	case BlocksBySliceMsg:
		return pm.handleBlocksBySlice(p, msg)
	case GetBlockByHashMsg:
		return pm.handleGetBlockByHash(p, msg)
	case NewBlockHashMsg:
		return pm.handleNewBlockAnnounce(p, msg)
	case NewBlockMsg:
		return pm.handleNewBlocks(p, msg)

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
	p.Log().Trace("Handle GET-LAST-MAIN-TIMESLICE request")

	lastMainSlice := pm.mainChain.GetLastTempMainBlkSlice()
	p.Log().Trace("Send timeslice back to remote node", "timeslice", lastMainSlice)
	return p.SendTimeSlice(lastMainSlice)
}

// 获取最近一次临时主块所在时间片消息处理
func (pm *ProtocolManager) handleLastMainTimeSlice(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process the last main timeslice response.")
	var timeslice uint64
	err := msg.Decode(&timeslice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Trace("Handle GET-LAST-MAIN-TIMESLICE response", "timeslice", timeslice)
	// 将回复结果递送到同步器
	return pm.synchroniser.DeliverLastTimeSliceResp(p.id, timeslice)
}

// 根据时间片获取对应所有区块hash消息处理
func (pm *ProtocolManager) handleGetBlockHashBySlice(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process block hash query.")
	//lastMainSlicer := pm.mainChain.GetLastTempMainBlkSlice()
	var targetSlice uint64 = 0
	err := msg.Decode(&targetSlice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Trace("<< Handle GET-BLOCK-HASH-BY-TIMESLICE request", "timeslice", targetSlice)

	var hashes []common.Hash
	hashes, err = pm.blkstorage.GetBlockHashByTmSlice(targetSlice)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug(">> Send hash back to remote node", "response-hash-size", len(hashes))
	return p.SendBlockHashes(targetSlice, hashes)
}

// 根据时间片获取对应所有区块hash消息处理
func (pm *ProtocolManager) handleBlockHashBySlice(p *peer, msg p2p.Msg) error {

	var response GetBlockHashBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Trace("<< Handle GET-BLOCK-HASH-BY-TIMESLICE response", "timeslice", response.Timeslice, "size", len(response.Hashes))
	// 将回复结果递送到同步器

	return pm.synchroniser.DeliverBlockHashesResp(p.id, response.Timeslice, response.Hashes)
}

// 根据区块hash返回对应区块（字节流）消息处理
func (pm *ProtocolManager) handleGetBlocksBySlice(p *peer, msg p2p.Msg) error {

	var req GetBlockDataBySliceReq
	err := msg.Decode(&req)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	if len(req.Hashes) <= 0 {
		log.Trace("Param 'Hashes' is empty.")
		return nil
	}
	p.Log().Trace("<< Handle GET-BLOCK-BY-SLICEHASH request", "timeslice", req.Timeslice, "size", len(req.Hashes))
	var blocks [][]byte
	blocks, err = pm.blkstorage.GetBlocks(req.Hashes)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if len(blocks) <= 0 {
		return nil
	}
	// 将结果回复给对方
	//p.Log().Trace("Handle GET-BLOCK-BY-SLICEHASH request", "timeslice", req.Timeslice, "size", len(req.Hashes))
	p.Log().Debug(">> Send block back to remote node", "response-block-size", len(blocks))
	return p.SendSliceBlocks(req.Timeslice, blocks)
}

func (pm *ProtocolManager) handleBlocksBySlice(p *peer, msg p2p.Msg) error {

	var response GetBlockDataBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Trace("<< handle GET-BLOCK-BY-TIMESLICE response", "timeslice", response.Timeslice, "size", len(response.Blocks))
	// 将回复结果递送到同步器
	return pm.synchroniser.DeliverBlockDatasResp(p.id, response.Timeslice, response.Blocks)
}

func (pm *ProtocolManager) handleGetBlockByHash(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process block query by hash.")
	var req []common.Hash
	err := msg.Decode(&req)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}

	if len(req) <= 0 {
		log.Trace("Param 'Hashes' is empty.")
		return nil
	}
	for _, item := range req {
		p.Log().Debug("<< Handle GET-BLOCK-BY-HASH request", "hash", item.String())
	}
	var blocks [][]byte
	blocks, err = pm.blkstorage.GetBlocks(req)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	if len(blocks) <= 0 {
		return nil
	}
	// 将结果回复给对方
	p.Log().Debug(">> Send block back to remote node", "response-block-size", len(blocks))
	return p.SendNewBlocks(blocks)
}

func (pm *ProtocolManager) handleNewBlocks(p *peer, msg p2p.Msg) error {

	var response [][]byte
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< Handle NEW-BLOCK response/announce", "size", len(response))
	return pm.synchroniser.DeliverNewBlockResp(p.id, response)
}

func (pm *ProtocolManager) handleNewBlockAnnounce(p *peer, msg p2p.Msg) error {

	var response common.Hash
	err := msg.Decode(&response)
	if err != nil {
		return errResp(ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< Handle NEW-BLOCK-HASH,new block announced", "hash", response.String())
	pm.synchroniser.MarkAnnounced(response, p.NodeID())
	return nil
}

/* func (pm *ProtocolManager) relayBlock(event *core.RelayBlocksEvent) error {
	log.Trace("Relay block")
	for _, p := range pm.peers.Peers() {
		for _, block := range event.Blocks {
			p.AsyncSendBlock(block.GetRlp())
		}
	}
	return nil
}

func (pm *ProtocolManager) getBlock(event *core.GetBlocksEvent) error {
	log.Trace("Process block query")
	for _, hash := range event.Hashes {
		pm.synchroniser.AsyncRequestBlock(hash)
	}
	return nil
}
*/
