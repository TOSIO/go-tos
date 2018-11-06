package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/sdag/core/protocol"
)

type TimeslicePacket struct {
	peerId    string
	timeslice uint64
}

type TSHashesPacket struct {
	peerId    string
	timeslice uint64
	hashes    []common.Hash
}

type TSBlocksPacket struct {
	peerId    string
	timeslice uint64
	blocks    [][]byte
}

type NewBlockPacket struct {
	peerId string
	blocks [][]byte
}

type SYNCBlockReqPacket struct {
	peerID     string
	beginPoint *protocol.TimesliceIndex
}

type SYNCBlockRespPacket struct {
	peerID   string
	response *protocol.SYNCBlockResponse
}

type SYNCBlockResACKPacket struct {
	peerID   string
	response *protocol.SYNCBlockResponseACK
}

/* type SYNCBlockEndPacket struct {
	peerID   string
	response *protocol.SYNCBlockEnd
}
*/
func (ts *TimeslicePacket) NodeID() string {
	return ts.peerId
}

func (ts *TimeslicePacket) ItemCount() int {
	return 1
}

func (bhs *TSHashesPacket) NodeID() string {
	return bhs.peerId
}

func (bhs *TSHashesPacket) ItemCount() int {
	return len(bhs.hashes)
}

func (bds *TSBlocksPacket) NodeID() string {
	return bds.peerId
}

func (bds *TSBlocksPacket) ItemCount() int {
	return len(bds.blocks)
}

func (bds *NewBlockPacket) NodeID() string {
	return bds.peerId
}

func (bds *NewBlockPacket) ItemCount() int {
	return 1
}

func (p *SYNCBlockRespPacket) NodeID() string {
	return p.peerID
}

func (p *SYNCBlockRespPacket) ItemCount() int {
	if p.response != nil {
		return len(p.response.TSBlocks)
	}
	return 0
}

func (p *SYNCBlockResACKPacket) NodeID() string {
	return p.peerID
}

func (p *SYNCBlockResACKPacket) ItemCount() int {
	return 0
}

func (p *SYNCBlockReqPacket) NodeID() string {
	return p.peerID
}

func (p *SYNCBlockReqPacket) ItemCount() int {
	return 0
}

/* func (p *SYNCBlockEndPacket) NodeID() string {
	return p.peerID
}

func (p *SYNCBlockEndPacket) ItemCount() int {
	return 0
}
*/
