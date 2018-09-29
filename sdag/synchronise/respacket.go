package synchronise

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

type TimeSliceResp struct {
	peerId    string
	timeSlice uint64
}

type SliceBlkHashesResp struct {
	peerId    string
	Timeslice uint64
	Hashes    []common.Hash
}

type BlockDatasResp struct {
	peerId string
	blocks [][]byte
}

func (ts *TimeSliceResp) NodeId() string {
	return ts.peerId
}

func (ts *TimeSliceResp) Items() int {
	return 1
}

func (bhs *SliceBlkHashesResp) NodeId() string {
	return bhs.peerId
}

func (bhs *SliceBlkHashesResp) Items() int {
	return len(bhs.Hashes)
}

func (bds *BlockDatasResp) NodeId() string {
	return bds.peerId
}

func (bds *BlockDatasResp) Items() int {
	return len(bds.blocks)
}
