package sdag

import (
	"sync"

	"github.com/TOSIO/go-tos/sdag/manager"

	"github.com/TOSIO/go-tos/sdag/mainchain"

	"github.com/TOSIO/go-tos/sdag/synchronise"

	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/internal/tosapi"
	"github.com/TOSIO/go-tos/node"
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

	synchroniser *synchronise.Synchroniser

	APIBackend    *SdagAPIBackend
	netRPCService *tosapi.PublicNetAPI

	// DB interfaces
	chainDb tosdb.Database // Block chain database

	eventMux       *interface{}
	accountManager *interface{}

	miner *interface{} //miner

	networkID uint64

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

// New creates a new Ethereum object (including the
// initialisation of the common Ethereum object)
func New(ctx *node.ServiceContext, config *Config) (*Sdag, error) {
	chainDb, err := CreateDB(ctx, config, "tos/db/sdagData/test")
	if err != nil {
		return nil, err
	}

	var chain mainchain.MainChainI
	if chain, err = mainchain.New(); err != nil {
		log.Error("Initialising Sdag blockchain failed.")
		return nil, err
	}
	protocolManager, err := NewProtocolManager(nil, config.NetworkId, chain, chainDb)
	if err != nil {
		log.Error("Initialising Sdag protocol failed.")
		return nil, err
	}
	protocolManager.Start(100)

	sdag := &Sdag{
		//初始化
		config:          config,
		shutdownChan:    make(chan bool),
		networkID:       config.NetworkId,
		chainDb:         chainDb,
		protocolManager: protocolManager,
		blockchain:      chain,
	}

	manager.SetDB(sdag.chainDb)
	manager.SetProtocolManager(sdag.protocolManager)

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
	s.netRPCService = tosapi.NewPublicNetAPI(srvr, s.NetVersion())
	return nil
}

func (s *Sdag) Stop() error {
	log.Debug("Sdag.Stop() called.")
	return nil
}

func (s *Sdag) SdagVersion() int   { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Sdag) NetVersion() uint64 { return s.networkID }

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
