package sdag

import (
	"encoding/json"
	"sync"

	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/manager"
	"github.com/TOSIO/go-tos/sdag/miner"

	"github.com/TOSIO/go-tos/sdag/mainchain"

	"github.com/TOSIO/go-tos/sdag/synchronise"

	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/internal/tosapi"
	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/services/p2p"
	"github.com/TOSIO/go-tos/services/rpc"
)

/*
sdag的账户、内存交易池、主链、数据库、同步、网络消息协议等各子模块在此管理（创建、初始化、子模块对象获取等）
*/

// Sdag implements the Ethereum full node service.
type Sdag struct {
	config *Config

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Ethereum

	// Handlers(tx mempool)
	blockchain      mainchain.MainChainI
	protocolManager *ProtocolManager //消息协议管理器（与p2p对接）

	queryBlockInfo *manager.QueryBlockInfoInterface //查询区块状态接口

	networkFeed *event.Feed

	blockPoolEvent *event.TypeMux

	blockPool *manager.BlockPool

	miner        *miner.Miner
	synchroniser *synchronise.Synchroniser
	//mempool       *manager.MemPool
	APIBackend    *SdagAPIBackend
	netRPCService *tosapi.PublicNetAPI

	// DB interfaces
	chainDb tosdb.Database // Block chain database

	//statedb
	stateDb state.Database //mpt trie

	eventMux       *interface{}
	accountManager *interface{}

	networkID uint64

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *Config) (*Sdag, error) {
	chainDB, err := CreateDB(ctx, config, "tos/db/sdagData/test")
	if err != nil {
		return nil, err
	}
	stateBaseDB, err := CreateDB(ctx, config, "tos/db/sdagData/stateDB")
	if err != nil {
		return nil, err
	}
	stateDB := state.NewDatabase(stateBaseDB)

	netFeed := new(event.Feed)
	var chain mainchain.MainChainI
	if chain, err = mainchain.New(chainDB, stateDB); err != nil {
		log.Error("Initialising Sdag blockchain failed.")
		return nil, err
	}
	event := &event.TypeMux{}
	protocolManager, err := NewProtocolManager(nil, config.NetworkId, chain, chainDB, netFeed, event)
	if err != nil {
		log.Error("Initialising Sdag protocol failed.")
		return nil, err
	}

	pool := manager.New(chain, chainDB, event)
	minerParam := miner.MinerInfo{}

	sdag := &Sdag{
		//初始化
		config:          config,
		shutdownChan:    make(chan bool),
		networkID:       config.NetworkId,
		chainDb:         chainDB,
		stateDb:         stateDB,
		protocolManager: protocolManager,
		blockchain:      chain,
		networkFeed:     netFeed,
		miner:           miner.New(pool, &minerParam, chain, netFeed),
		blockPool:       pool,
		blockPoolEvent:  event,
	}

	log.Info("Initialising Sdag protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	sdag.APIBackend = &SdagAPIBackend{sdag}

	return sdag, nil
}

//实现service APIs()接口,返回sdag支持的rpc API接口
func (s *Sdag) APIs() []rpc.API {
	log.Debug("Sdag.APIs() called.")

	apis := tosapi.GetAPIs(s.APIBackend)

	// Append all the local APIs and return
	return append(apis, []rpc.API{
		{
			Namespace: "sdag",
			Version:   "1.0",
			Service:   NewPublicSdagAPI(s),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
//实现service Protocols()接口,返回sdag支持的rpc API接口
func (s *Sdag) Protocols() []p2p.Protocol {
	log.Debug("Sdag.Protocols() called.")
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// tos protocol implementation.
//实现service Start()接口,启动协议
func (s *Sdag) Start(srvr *p2p.Server) error {
	log.Debug("Sdag.Start() called.")
	// Start the RPC service
	//s.mempool.Start()
	s.protocolManager.Start(100)
	s.miner.Start()
	s.netRPCService = tosapi.NewPublicNetAPI(srvr, s.NetVersion())
	return nil
}

func (s *Sdag) Stop() error {
	log.Debug("Sdag.Stop() called.")
	s.protocolManager.Stop()
	s.miner.Stop()
	return nil
}

func (s *Sdag) SdagVersion() int   { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Sdag) NetVersion() uint64 { return s.networkID }

func (s *Sdag) BlockPool() core.BlockPoolI {
	return s.blockPool
}

func (s *Sdag) BlockPoolEvent() *event.TypeMux {
	return s.blockPoolEvent
}

func (s *Sdag) Status() string {
	status := s.protocolManager.GetStatus()
	data, err := json.Marshal(status)
	if err != nil {
		return ""
	} else {
		return string(data)
	}
}

// CreateDB creates the chain database.
func CreateDB(ctx *node.ServiceContext, config *Config, name string) (tosdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*tosdb.LDBDatabase); ok {
		db.Meter("tos/db/sdagData/")
	}
	return db, nil
}
func (s *Sdag) SdagNodeIDMessage() []string {
	var NodeIdMessage []string
	NodeIdMessage = s.protocolManager.RealNodeIdMessage()
	return NodeIdMessage
}
