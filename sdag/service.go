package sdag

import (
	"fmt"
	"github.com/TOSIO/go-tos/sdag/transaction"
	"sync"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/services/accounts"
	"github.com/TOSIO/go-tos/services/p2p/discover"

	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/manager"
	"github.com/TOSIO/go-tos/sdag/miner"

	"github.com/TOSIO/go-tos/sdag/mainchain"

	"github.com/TOSIO/go-tos/sdag/synchronise"

	"net"

	"github.com/TOSIO/go-tos/devbase/event"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/internal/tosapi"
	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/sdag/core/protocol"
	"github.com/TOSIO/go-tos/services/messagequeue"
	"github.com/TOSIO/go-tos/services/p2p"
	"github.com/TOSIO/go-tos/services/rpc"
)

// Sdag implements the node service.
type Sdag struct {
	config *Config

	// Channel for shutting down the service
	shutdownChan chan bool // Channel for shutting down the Ethereum

	// Handlers(tx mempool)
	blockchain      mainchain.MainChainI
	protocolManager *ProtocolManager // protocol manager of p2p

	queryBlockInfo *manager.QueryBlockInfoInterface // query the interface status of blcok

	networkFeed *event.Feed

	blockPoolEvent *event.TypeMux

	blockPool   *manager.BlockPool
	transaction *transaction.Transaction

	miner        *miner.Miner
	synchroniser *synchronise.Synchroniser
	//mempool       *manager.MemPool
	APIBackend    *SdagAPIBackend
	netRPCService *tosapi.PublicNetAPI

	// DB interfaces
	chainDb tosdb.Database // Block chain database

	eventMux       *interface{}
	accountManager *accounts.Manager
	tosbase        common.Address

	networkID uint64
	nodeID    string

	lock sync.RWMutex
	sct  *node.ServiceContext

	mq *messagequeue.MessageQueue

	// Protects the variadic fields (e.g. gas price and etherbase)
}

func New(ctx *node.ServiceContext, config *Config) (*Sdag, error) {
	chainDB, err := CreateDB(ctx, config, "levelDB/sdagData/chain")
	if err != nil {
		return nil, err
	}
	stateBaseDB, err := CreateDB(ctx, config, "levelDB/sdagData/state")
	if err != nil {
		return nil, err
	}

	var mq *messagequeue.MessageQueue
	if config.MessageQueue {
		mq, err = messagequeue.CreateMQ()
		if err != nil {
			log.Error(err.Error())
		}
	}
	netFeed := new(event.Feed)
	var chain mainchain.MainChainI
	if chain, err = mainchain.New(chainDB, stateBaseDB, config.VMConfig, config.NetworkId, mq); err != nil {
		log.Error("Initialising Sdag blockchain failed.")
		return nil, err
	}
	event := &event.TypeMux{}
	protocolManager, err := NewProtocolManager(nil, config.NetworkId, chain, chainDB, netFeed, event)
	if err != nil {
		log.Error("Initialising Sdag protocol failed.")
		return nil, err
	}
	pool := manager.New(chain, chainDB, netFeed, event, mq)

	sdag := &Sdag{
		// init
		config:          config,
		shutdownChan:    make(chan bool),
		networkID:       config.NetworkId,
		chainDb:         chainDB,
		protocolManager: protocolManager,
		blockchain:      chain,
		networkFeed:     netFeed,
		miner:           miner.New(pool, &miner.MinerInfo{}, chain, netFeed),
		accountManager:  ctx.AccountManager,
		tosbase:         config.Tosbase,
		blockPool:       pool,
		transaction:     transaction.New(pool, chain, chainDB),
		blockPoolEvent:  event,
		sct:             ctx,
		mq:              mq,
	}

	log.Info("Initialising Sdag protocol", "versions", protocol.ProtocolVersions, "network", config.NetworkId)

	sdag.APIBackend = &SdagAPIBackend{sdag}

	return sdag, nil
}

// APIs implement the interface of service APIs and return rpc API interface
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
func (s *Sdag) Protocols() []p2p.Protocol {
	log.Debug("Sdag.Protocols() called.")
	return s.protocolManager.SubProtocols
}

// Start implements node.Service, starting all internal goroutines needed by the
// tos protocol implementation.
func (s *Sdag) Start(srv *p2p.Server) error {
	log.Debug("Sdag.Start() called.")
	// Start the RPC service
	//s.mempool.Start()
	s.protocolManager.Start(100)
	// Configure the local mining address
	//eb, err := s.Tosbase()
	//if err != nil {
	//	log.Debug("Cannot start mining without tosbase", "err", err)
	//	//return fmt.Errorf("tosbase missing: %v", err)
	//}
	s.nodeID = discover.PubkeyID(&srv.Config.PrivateKey.PublicKey).String()
	//if s.config.Mining{
	//	log.Debug("Cannot start mining ", "configMining", s.config.Mining)
	//	s.miner.Start(eb)
	//}
	s.miner.StartInit()
	s.netRPCService = tosapi.NewPublicNetAPI(srv, s.NetVersion())
	return nil
}

func (s *Sdag) Stop() error {
	log.Debug("Sdag.Stop() called.")
	s.protocolManager.Stop()
	s.miner.Stop()
	log.Debug("SDAG was stopped")
	return nil
}

func (s *Sdag) SdagVersion() int                  { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *Sdag) NetVersion() uint64                { return s.networkID }
func (s *Sdag) AccountManager() *accounts.Manager { return s.accountManager }

func (s *Sdag) BlockPool() core.BlockPoolI {
	return s.blockPool
}

func (s *Sdag) BlockPoolEvent() *event.TypeMux {
	return s.blockPoolEvent
}

func (s *Sdag) Status() status {
	return s.protocolManager.GetStatus()
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
func (s *Sdag) SdagNodeIDMessage() ([]string, int) {
	var NodeIdMessage []string
	var ConnectNumber int
	NodeIdMessage, ConnectNumber = s.protocolManager.RealNodeIdMessage()
	return NodeIdMessage, ConnectNumber
}

func (s *Sdag) Tosbase() (eb common.Address, err error) {
	s.lock.RLock()
	tosbase := s.tosbase
	s.lock.RUnlock()
	if tosbase != (common.Address{}) {
		return tosbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			tosbase := accounts[0].Address

			s.lock.Lock()
			s.tosbase = tosbase
			s.lock.Unlock()

			log.Info("Tosbase automatically configured", "address", tosbase)
			return tosbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("tosbase must be explicitly specified")
}

func (s *Sdag) LocalNodeIP() (string, bool) {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		fmt.Println("net.Interfaces failed, err:", err.Error())
		return "", false
	}

	for i := 0; i < len(netInterfaces); i++ {
		if (netInterfaces[i].Flags & net.FlagUp) != 0 {
			addrs, _ := netInterfaces[i].Addrs()

			for _, address := range addrs {
				if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						return ipnet.IP.String(), true
					}
				}
			}
		}
	}
	return "", false
}
