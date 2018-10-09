package sdag

import (
	"github.com/TOSIO/go-tos/devbase/common"
)

// Constants to match up protocol versions and messages
const (
	tos62 = 62
	tos63 = 63
)

// ProtocolName is the official short name of the protocol used during capability negotiation.
var ProtocolName = "tos"

// ProtocolVersions are the upported versions of the tos protocol (first is primary).
var ProtocolVersions = []uint{tos63, tos62}

// ProtocolLengths are the number of implemented message corresponding to different protocol versions.
var ProtocolLengths = []uint64{17, 8}

const ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message

// tos protocol message codes
const (
	// Protocol messages belonging to tos/62
	StatusMsg         = 0x00
	NewBlockMsg       = 0x01
	NewBlockHashMsg   = 0x02
	GetBlockByHashMsg = 0x03

	GetLastMainTimeSlice   = 0x04
	LastMainTimeSlice      = 0x05
	GetBlockHashBySliceMsg = 0x06
	BlockHashBySliceMsg    = 0x07
	GetBlocksBySliceMsg    = 0x08
	BlocksBySliceMsg       = 0x09

	// Protocol messages belonging to tos/63
	GetNodeDataMsg = 0x0d
	NodeDataMsg    = 0x0e
	GetReceiptsMsg = 0x0f
	ReceiptsMsg    = 0x10
)

type GetBlockHashBySliceResp struct {
	Timeslice uint64
	Hashes    []common.Hash
}

type GetBlockDataBySliceReq struct {
	Timeslice uint64
	Hashes    []common.Hash
}

type GetBlockDataBySliceResp struct {
	Timeslice uint64
	Blocks    [][]byte
}

type errCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
)

func (e errCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
}
