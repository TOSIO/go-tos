package sdag

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/services/p2p"
)

type ProtocolManager struct {
	networkID uint64

	maxPeers int

	peers *interface{}

	SubProtocols []p2p.Protocol

	// channels for fetcher, syncer, txsyncLoop
	newPeerCh   chan *interface{}
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
	manager := &ProtocolManager{}

	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {
	fmt.Println("ProtocolManager.removePeer called.")
}

func (pm *ProtocolManager) Start(maxPeers int) {
	fmt.Println("ProtocolManager.Start called.")

}

func (pm *ProtocolManager) Stop() {
	fmt.Println("ProtocolManager.Stop called.")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *interface{} {
	fmt.Println("ProtocolManager.newPeer called.")
	return nil
}

func (pm *ProtocolManager) handle(p *interface{}) error {
	fmt.Println("ProtocolManager.handle called.")
	return nil
}

func (pm *ProtocolManager) handleMsg(p *interface{}) error {
	fmt.Println("ProtocolManager.handleMsg called.")
	return nil
}

// NodeInfo retrieves some protocol metadata about the running host node.
func (pm *ProtocolManager) NodeInfo() *NodeInfo {
	fmt.Println("ProtocolManager.NodeInfo called.")
	return nil
}
