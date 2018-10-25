package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
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
