package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

type TimeSlicePacket struct {
	peerId    string
	timeSlice uint64
}

type SliceBlkHashesPacket struct {
	peerId    string
	timeslice uint64
	hashes    []common.Hash
}

type SliceBlkDatasPacket struct {
	peerId    string
	timeslice uint64
	blocks    [][]byte
}

func (ts *TimeSlicePacket) NodeId() string {
	return ts.peerId
}

func (ts *TimeSlicePacket) Items() int {
	return 1
}

func (bhs *SliceBlkHashesPacket) NodeId() string {
	return bhs.peerId
}

func (bhs *SliceBlkHashesPacket) Items() int {
	return len(bhs.hashes)
}

func (bds *SliceBlkDatasPacket) NodeId() string {
	return bds.peerId
}

func (bds *SliceBlkDatasPacket) Items() int {
	return len(bds.blocks)
}
