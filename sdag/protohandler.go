package sdag

import (
	"errors"
	"fmt"
	"strconv"

	//"github.com/TOSIO/go-tos/sdag/core/storage"
	"math/big"
	"sync"

	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/protocol"

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
	NodeNum  int                  `json:"node_num"`
	Status   int                  `json:"status"` //0-none,1-syncing,2-ready,3-working,4-connecting
	Syncstat core.SYNCStatusEvent `json:"sync_status,omitempty"`
}

// ProtocolManager,sdag Protocol management and implementation
type ProtocolManager struct {
	networkID uint64

	maxPeers int

	peers *peerSet

	blockPoolEvent *event.TypeMux
	relaySub       *event.TypeMuxSubscription
	getSub         *event.TypeMuxSubscription
	getIsolateSub  *event.TypeMuxSubscription

	syncEvent   *event.TypeMux
	syncstatSub *event.TypeMuxSubscription

	networkFeed *event.Feed
	feeded      bool

	chainDb tosdb.Database // Block chain database

	//blockChain   mainchain.MainChainI
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

func (pm *ProtocolManager) RealNodeIdMessage() ([]string, int) {

	var peerId = make([]string, 0)
	var ConnectNumber = 0
	peerSetMessage := pm.peers
	peerMessage := peerSetMessage.peers
	for _, peerIdMessage := range peerMessage {
		//fmt.Println("IDorIP:  ", peerIdMessage.id, peerIdMessage.RemoteAddr().String())
		ConnectNumber = ConnectNumber + 1
		peerId = append(peerId, "ID:"+peerIdMessage.id, "IP:"+peerIdMessage.RemoteAddr().String(), "------")
	}

	return peerId, ConnectNumber

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

		mainChain:      chain,
		networkFeed:    feed,
		blockPoolEvent: poolFeed,
		feeded:         false,
		stat:           status{NodeNum: 0, Status: STAT_NONE},
		syncResult:     make(chan error),
	}

	manager.relaySub = manager.blockPoolEvent.Subscribe(&core.RelayBlocksEvent{})
	manager.getSub = manager.blockPoolEvent.Subscribe(&core.GetNetworkNewBlocksEvent{})
	manager.getIsolateSub = manager.blockPoolEvent.Subscribe(&core.GetIsolateBlocksEvent{})

	manager.syncEvent = &event.TypeMux{}
	manager.syncstatSub = manager.syncEvent.Subscribe(core.SYNCStatusEvent{})
	// Initiate a sub-protocol for every implemented version we can handle
	var err error
	if err = manager.initProtocols(); err != nil {
		return nil, err
	}
	//var storageProxy *synchronise.StorageProxy
	//var mempoolProxy *synchronise.MemPoolProxy
	if manager.blkstorage, err = synchronise.NewStorage(db); err != nil {
		log.Error("Initialising Sdag storageproxy failed.")
		return nil, err
	}
	/* if mempoolProxy, err = synchronise.NewMempol(); err != nil {
		log.Error("Initialising Sdag mempool failed.")
		return nil, err
	} */

	if manager.synchroniser, err = synchronise.NewSynchroinser(manager.Peers(),
		chain, manager.blkstorage, feed, poolFeed, manager.syncEvent); err != nil {
		log.Error("Initialising Sdag synchroniser failed.")
		return nil, err
	}

	return manager, nil
}

// loop ,this function will be removed in future
func (pm *ProtocolManager) loop() {
	for {
		select {
		case ev := <-pm.relaySub.Chan():
			log.Debug("pm.relaySub.Chan 1")
			if event, ok := ev.Data.(*core.RelayBlocksEvent); ok {
				for _, block := range event.Blocks {
					pm.synchroniser.Broadcast(block.GetHash())
				}
			}
			log.Debug("pm.relaySub.Chan 2")
		case ev := <-pm.getSub.Chan():
			log.Debug("pm.getSub.Chan 1")
			if event, ok := ev.Data.(*core.GetNetworkNewBlocksEvent); ok {
				for _, block := range event.Hashes {
					//log.Debug("Request", "peer.id", peer.NodeID)
					pm.synchroniser.RequestBlock(block)
				}
			}
			log.Debug("pm.getSub.Chan 2")
		case ev := <-pm.getIsolateSub.Chan():
			log.Debug("pm.getIsolateSub.Chan 1")
			if event, ok := ev.Data.(*core.GetIsolateBlocksEvent); ok {
				for _, block := range event.Hashes {
					//log.Debug("Request", "peer.id", peer.NodeID)
					pm.synchroniser.RequestIsolatedBlock(block)
				}
			}
			log.Debug("pm.getIsolateSub.Chan 2")
		case peer := <-pm.newPeerCh:
			// Make sure we have peers to select from, then sync
			log.Debug("Accept a new peer,", "peer.id", peer.NodeID())
		case <-pm.noMorePeers:
			log.Debug("Exist for NO-MORE-PEERS")
			return
		case ev := <-pm.syncstatSub.Chan():
			log.Debug("pm.syncstatSub.Chan 1")
			if event, ok := ev.Data.(core.SYNCStatusEvent); ok {
				if event.Progress == core.SYNC_END && event.Err == nil {
					pm.stat.Status = STAT_WORKING
					pm.stat.Syncstat = core.SYNCStatusEvent{}
				}
				pm.stat.Syncstat = event
				log.Debug("Synchronizing", "progress", event.Progress.String(), "curorigin", event.CurOrigin, "curTS", event.CurTS, "curIndex(TS)", event.Index,
					"startTS", event.BeginTS,
					"endTS(cur)", event.EndTS,
					"beginTime", event.BeginTime,
					"endTime", event.EndTime,
					"accumlatedNum", event.AccumulateSYNCNum,
					//"tiredOrigins", event.TriedOrigin,
					"err", event.Err)
			}
			log.Debug("pm.syncstatSub.Chan 2")
		}
	}
}

func (pm *ProtocolManager) initProtocols() error {
	pm.SubProtocols = make([]p2p.Protocol, 0, len(protocol.ProtocolVersions))
	for i, version := range protocol.ProtocolVersions {

		// Compatible; initialise the sub-protocol
		version := version // Closure for the run
		pm.SubProtocols = append(pm.SubProtocols, p2p.Protocol{
			Name:    protocol.ProtocolName,
			Version: version,
			Length:  protocol.ProtocolLengths[i],
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
		log.Debug("Disconnect peer", "peer", id)
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
	pm.synchroniser.Clear(id)

	if pm.peers.Len() <= 0 {
		log.Debug("Post connection close event")
		pm.networkFeed.Send(core.NETWORK_CLOSED)
		log.Debug("Post connection close event completed")
		pm.feeded = false
		pm.stat.Status = STAT_NET_UNVAILABLE
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	log.Info("ProtocolManager.Start called.")
	// start sync procedure
	pm.stat.Status = STAT_READY

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

	genesis, err := pm.mainChain.GetGenesisHash()
	if err != nil {
		return err
	}
	//firstMBTimeslice := uint64(0)
	/* first, _, err := pm.mainChain.GetNextMain(genesis)
	if err == nil {
		//return err
		firstMBlock := pm.blkstorage.GetBlock(first)
		if firstMBlock != nil {
			firstMBTimeslice = utils.GetMainTime(firstMBlock.GetTime())
		} else {
			p.Log().Debug("Failed to get the first main-block")
		}
	} else {
		genesisBlock := pm.blkstorage.GetBlock(genesis)
		if genesisBlock != nil {
			firstMBTimeslice = utils.GetMainTime(genesisBlock.GetTime())
		} else {
			p.Log().Debug("Failed to get the genesis-block")
		}
	} */

	lastTmpMBTimeslice := pm.mainChain.GetLastTempMainBlkSlice()
	if err := p.Handshake(pm.networkID, genesis /* firstMBTimeslice,  */, lastTmpMBTimeslice,
		pm.mainChain.GetMainTail().Number, pm.mainChain.GetMainTail().CumulativeDiff); err != nil {
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
	if !pm.feeded {
		pm.networkFeed.Send(core.NETWORK_CONNECTED)
		pm.feeded = true
		p.Log().Debug("Post event to miner")
	}
	defer pm.removePeer(p.id)

	p.Log().Debug("TOS handshake is done", "Node.LastTempMBTS", p.lastTempMBTimeslice, "local.LastTempMBTS", lastTmpMBTimeslice,
		"node.LastMBNum", p.lastMainBlockNum, "local.LastMBNum", pm.mainChain.GetMainTail().Number,
		"node.Diff", p.lastCumulatedDiff.String(), "local.Diff", pm.mainChain.GetMainTail().CumulativeDiff.String())

	if p.lastTempMBTimeslice > lastTmpMBTimeslice && p.lastMainBlockNum > pm.mainChain.GetMainTail().Number {
		pm.stat.Status = STAT_SYNCING
		pm.syncEvent.Post(&core.NewSYNCTask{
			NodeID:            p.NodeID(),
			LastCumulatedDiff: *p.lastCumulatedDiff,
			LastMainBlockNum:  p.lastMainBlockNum,
			//FirstMBTimeslice:    p.firstMBTimeslice,
			LastTempMBTimeslice: p.lastTempMBTimeslice,
		})
		p.Log().Debug("Post SYNCTask", "lastCumulatedDiff", p.lastCumulatedDiff, "lastMainBlockNum", p.lastMainBlockNum, "lastTS", p.lastTempMBTimeslice)
	} else {
		if pm.stat.Status != STAT_SYNCING {
			pm.stat.Status = STAT_WORKING
		}
	}
	// main loop. handle incoming messages.
	for {
		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("TOS message handling failed", "err", err)
			return err
		}
	}
	//p.Log().Debug("TOS message handling exit")
}
func errResp(code protocol.ErrCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

func (pm *ProtocolManager) GetStatus() status {
	pm.stat.NodeNum = pm.peers.Len()
	return pm.stat
}

func (pm *ProtocolManager) handleMsg(p *peer) error {
	//p.Log().Info("Starting handle message")
	//p.Log().Debug("handleMsg 1")
	//defer p.Log().Debug("handleMsg 2")
	msg, err := p.rw.ReadMsg()
	if err != nil {
		log.Debug("Error handle message", "err", err)
		return err
	}
	if msg.Size > protocol.ProtocolMaxMsgSize {
		return errResp(protocol.ErrMsgTooLarge, "%v > %v", msg.Size, protocol.ProtocolMaxMsgSize)
	}
	defer msg.Discard()
	p.Log().Debug("Handle message", "msg.Code", protocol.MsgCodeToString(int(msg.Code)))
	//dispatch message here
	switch msg.Code {
	case protocol.StatusMsg:
		// Status messages should never arrive after the handshake
		return errResp(protocol.ErrExtraStatusMsg, "uncontrolled status message")
	case protocol.GetLastMainTimeSlice: // get the temporary time slice of the last main time
		return pm.handleGetLastMainTimeSlice(p, msg)
	case protocol.LastMainTimeSlice:
		return pm.handleLastMainTimeSlice(p, msg)
	case protocol.GetBlockHashBySliceMsg: // get all hashes by time slice
		return pm.handleGetBlockHashBySlice(p, msg)
	case protocol.BlockHashBySliceMsg:
		return pm.handleBlockHashBySlice(p, msg)
	case protocol.GetBlocksBySliceMsg: // get block data
		return pm.handleGetBlocksBySlice(p, msg)
	case protocol.SYNCBlockRequestMsg:
		return pm.handleSYNCblockRequest(p, msg)
	case protocol.SYNCBlockResponseMsg:
		return pm.handleSYNCblockResponse(p, msg)
	case protocol.SYNCBlockResponseACKMsg:
		return pm.handleSYNCblockResponseACK(p, msg)
	case protocol.BlocksBySliceMsg:
		return pm.handleBlocksBySlice(p, msg)
	case protocol.GetBlockByHashMsg:
		return pm.handleGetBlockByHash(p, msg)
	case protocol.NewBlockHashMsg:
		return pm.handleNewBlockAnnounce(p, msg)
	case protocol.NewBlockMsg:
		return pm.handleNewBlocks(p, msg)
	case protocol.GetlocatorRequestMsg: //handle getlocater
		return pm.handleGetlocatorRequest(p, msg)
	case protocol.LocatorResponseMsg:
		return pm.handleLocatorResponse(p, msg)
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

// handleGetLastMainTimeSlice send the main time slice of last temporary block  request
func (pm *ProtocolManager) handleGetLastMainTimeSlice(p *peer, msg p2p.Msg) error {
	p.Log().Debug("<< GET-LAST-MAINBLOCK-TIMESLICE")

	lastMainSlice := pm.mainChain.GetLastTempMainBlkSlice()
	p.Log().Debug(">> LAST-MAINBLOCK-TIMESLICE", "timeslice", lastMainSlice)
	return p.SendTimeSlice(lastMainSlice)
}

// handleLastMainTimeSlice implement the request
func (pm *ProtocolManager) handleLastMainTimeSlice(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process the last main timeslice response.")
	var timeslice uint64
	err := msg.Decode(&timeslice)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< LAST-MAINBLOCK-TIMESLICE", "timeslice", timeslice)
	// Deliver the response to the synchronizer
	return pm.synchroniser.DeliverLastTimeSliceResp(p.id, timeslice)
}

//handleGetBlockHashBySlice send the request
func (pm *ProtocolManager) handleGetBlockHashBySlice(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process block hash query.")
	//lastMainSlicer := pm.mainChain.GetLastTempMainBlkSlice()
	var targetSlice uint64 = 0
	err := msg.Decode(&targetSlice)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< GET-BLOCK-HASH-BY-TIMESLICE", "timeslice", targetSlice)

	var hashes []common.Hash
	hashes, err = pm.blkstorage.GetBlockHashByTmSlice(targetSlice)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}

	err = p.SendBlockHashes(targetSlice, hashes)
	p.Log().Debug(">> BLOCK-HASH-BY-TIMESLICE", "size", len(hashes), "err", err)
	return err
}

// handleBlockHashBySlice get blcok hash according to the specific time slice
func (pm *ProtocolManager) handleBlockHashBySlice(p *peer, msg p2p.Msg) error {

	var response protocol.GetBlockHashBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< BLOCK-HASH-BY-TIMESLICE", "timeslice", response.Timeslice, "size", len(response.Hashes))
	// Deliver the response to the synchronizer

	return pm.synchroniser.DeliverBlockHashesResp(p.id, response.Timeslice, response.Hashes)
}

// handleGetBlocksBySlice handle the request
func (pm *ProtocolManager) handleGetBlocksBySlice(p *peer, msg p2p.Msg) error {

	var req protocol.GetBlockDataBySliceReq
	err := msg.Decode(&req)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}

	if len(req.Hashes) <= 0 {
		log.Debug("Param 'Hashes' is empty.")
		return nil
	}
	p.Log().Debug("<< GET-BLOCK-BY-SLICEHASH", "timeslice", req.Timeslice, "size", len(req.Hashes))
	var blocks [][]byte
	blocks, err = pm.blkstorage.GetBlocks(req.Hashes)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	if len(blocks) <= 0 {
		return nil
	}
	//Deliver the response to the other
	//p.Log().Trace("Handle GET-BLOCK-BY-SLICEHASH request", "timeslice", req.Timeslice, "size", len(req.Hashes))
	p.Log().Debug(">> BLOCK-BY-SLICEHASH", "response-block-size", len(blocks))
	return p.SendSliceBlocks(req.Timeslice, blocks)
}

func (pm *ProtocolManager) handleBlocksBySlice(p *peer, msg p2p.Msg) error {

	var response protocol.GetBlockDataBySliceResp
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< BLOCK-BY-TIMESLICE", "timeslice", response.Timeslice, "size", len(response.Blocks))
	//Deliver the response to the synchronizer
	return pm.synchroniser.DeliverBlockDatasResp(p.id, response.Timeslice, response.Blocks)
}

func (pm *ProtocolManager) handleGetBlockByHash(p *peer, msg p2p.Msg) error {
	//p.Log().Trace("Process block query by hash.")
	var req []common.Hash
	err := msg.Decode(&req)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}

	if len(req) <= 0 {
		log.Debug("Param 'Hashes' is empty.")
		return nil
	}
	for _, item := range req {
		p.Log().Debug("<< GET-BLOCK-BY-HASH", "hash", item.String())
	}
	var blocks [][]byte
	blocks, err = pm.blkstorage.GetBlocks(req)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	if len(blocks) <= 0 {
		//forward
		for _, hash := range req {
			pm.synchroniser.RequestBlock(hash)
		}
		log.Debug("Forward request block", "size", len(req))
		return nil
	}
	// Deliver the response to the other
	p.Log().Debug(">> BLOCK-BY-HASH", "size", len(blocks))
	return p.SendNewBlocks(blocks)
}

func (pm *ProtocolManager) handleNewBlocks(p *peer, msg p2p.Msg) error {

	var response [][]byte
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< NEW-BLOCK(response/sync)", "size", len(response))
	return pm.synchroniser.DeliverNewBlockResp(p.id, response)
}

func (pm *ProtocolManager) handleNewBlockAnnounce(p *peer, msg p2p.Msg) error {
	if pm.synchroniser.ExceedAnnounceLimit(p.NodeID()) {
		return errors.New("block annouce is exceed limit")
	}
	var response common.Hash
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< NEW-BLOCK-HASH", "hash", response.String())
	pm.synchroniser.MarkAnnounced(response, p.NodeID())
	pm.blockPoolEvent.Post(&core.AnnounceEvent{Hash: response})
	return nil
}

func (pm *ProtocolManager) handleSYNCblockRequest(p *peer, msg p2p.Msg) error {
	var request protocol.SYNCBlockRequest
	err := msg.Decode(&request)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< SYNC-BLOCK-REQUEST", "timeslice", request.BeginPoint.Timeslice, "index", request.BeginPoint.Index)
	return pm.synchroniser.DeliverSYNCBlockRequest(p.id, &request.BeginPoint)
}

func (pm *ProtocolManager) handleSYNCblockResponse(p *peer, msg p2p.Msg) error {
	var response protocol.SYNCBlockResponse
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< SYNC-BLOCK-RESPONSE", "size", len(response.TSBlocks))
	return pm.synchroniser.DeliverSYNCBlockResponse(p.id, &response)
}

func (pm *ProtocolManager) handleSYNCblockResponseACK(p *peer, msg p2p.Msg) error {
	var response protocol.SYNCBlockResponseACK
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< SYNC-BLOCK-RESPONSE-ACK", "timeslice", response.ConfirmPoint.Timeslice, "index", response.ConfirmPoint.Index)
	return pm.synchroniser.DeliverSYNCBlockACKResponse(p.id, &response)
}

func (pm *ProtocolManager) handleGetlocatorRequest(p *peer, msg p2p.Msg) error {
	var request protocol.GetLocatorRequest
	err := msg.Decode(&request)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	log.Debug("handle get locator request", "peer", p.id)
	var samples []protocol.MainChainSample //define a sample block slice
	sample := protocol.MainChainSample{}
	numberEnd := pm.mainChain.GetMainTail().Number

	var count uint64 = 1
	var interval uint64 = 0
	log.Debug("Endnumber", "number", numberEnd)
	for i := 1; numberEnd > interval; i++ {
		log.Debug("Handling", "interval", interval)
		locatorNumber := numberEnd - interval
		numberEnd = locatorNumber
		interval = uint64(count << uint64(i))
		log.Debug("Handling next", "interval", interval)
		sample.Number = locatorNumber
		if sample.Hash, err = pm.blkstorage.GetMainBlock(locatorNumber); err != nil {
			log.Warn("get block hash by slice is error")

		} else {
			log.Debug("Ouput locator-sample", "number", sample.Number, "hash", sample.Hash.String())
			samples = append(samples, sample)
		}
	}
	var genesis common.Hash
	if genesis, err = pm.mainChain.GetGenesisHash(); err != nil {
		log.Warn("get genesis hash is error")
	}
	samples = append(samples, protocol.MainChainSample{Number: uint64(0), Hash: genesis})
	log.Debug("Ouput locator-sample", "number", uint64(0), "hash", genesis.String())

	//response packets to peer
	p.Log().Debug(">> response-locator-msg", "locator-size", len(samples))
	return p.SendLocatorPackge(samples)
}

func (pm *ProtocolManager) handleLocatorResponse(p *peer, msg p2p.Msg) error {
	var response []protocol.MainChainSample
	err := msg.Decode(&response)
	if err != nil {
		return errResp(protocol.ErrDecode, "msg %v: %v", msg, err)
	}
	p.Log().Debug("<< locator-response", "size", len(response))
	return pm.synchroniser.DeliverLocatorResponse(p.id, response)
	/*var beginTimeslice protocol.SampleBlockResponseMsg
	beginTimeslice.BeginPoint = 0
	for i := len(response) - 1; i >= 0; i-- {
		block := pm.blkstorage.GetBlock(response[i].Hash)
		if block != nil {
			timeslice := uint64(utils.GetMainTime(block.GetTime()))
			if timeslice == response[i].Timeslice {
				beginTimeslice.BeginPoint = timeslice
			}
		}
	}
	if beginTimeslice.BeginPoint == 0 {
		log.Debug("beginpoint msg", "beginpoint", beginTimeslice.BeginPoint)
		fmt.Printf("not locate begintimeslice")
	}
	log.Debug("handled response", "begintimeslice", beginTimeslice.BeginPoint)*/

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
func (pm *ProtocolManager) CalculateProgressPercent() string {

	var progressPercent string
	if pm.stat.Syncstat.Progress == 3 {
		progressPercent = "100"
	} else if pm.stat.Syncstat.CurTS == 0 {
		progressPercent = "0"
	} else if pm.stat.Syncstat.EndTS != pm.stat.Syncstat.BeginTS {
		progressPct := float64(pm.stat.Syncstat.CurTS-pm.stat.Syncstat.BeginTS) / float64(pm.stat.Syncstat.EndTS-pm.stat.Syncstat.BeginTS)
		progress := int(progressPct * 100)
		progressPercent = strconv.Itoa(progress)
	}
	return progressPercent + "%"
}
