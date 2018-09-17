package sdag

import (
	"sync"

	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/node"
	"github.com/TOSIO/go-tos/services/p2p"
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
	blockchain      *interface{}
	protocolManager *ProtocolManager //消息协议管理器（与p2p对接）

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
	sdag := &Sdag{
		//初始化
		config:       config,
		shutdownChan: make(chan bool),
		networkID:    config.NetworkId,
	}

	log.Info("Initialising Sdag protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	var err error
	if sdag.protocolManager, err = NewProtocolManager(nil, config.NetworkId); err != nil {
		return nil, err
	}

	return sdag, nil
}

//实现service APIs()接口,返回sdag支持的rpc API接口
func (s *Sdag) APIs() []interface{} {
	log.Debug("Sdag.APIs() called.")
	return nil
}

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
//实现service Protocols()接口,返回sdag支持的rpc API接口
func (s *Sdag) Protocols() []p2p.Protocol {
	log.Debug("Sdag.Protocols() called.")
	return nil
}

// Start implements node.Service, starting all internal goroutines needed by the
// tos protocol implementation.
//实现service Start()接口,启动协议
func (s *Sdag) Start(srvr *p2p.Server) error {
	log.Debug("Sdag.Start() called.")
	return nil
}

func (s *Sdag) Stop() error {
	log.Debug("Sdag.Stop() called.")
	return nil
}
