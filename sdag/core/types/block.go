package types

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
)

var (
	ErrInvalidSig = errors.New("invalid transaction v, r, s values")
)

// Difine some common interfaces of block

// Cet interface
// 1. get block hash
// 2. block difficulty,digging block and transaction block have some different，refering 6.1 block hash
// 3. cumulative difficulty
// 4. blcok time
// 5. the sender of the block,get the sender address according to vrs signature
// 6. block links
// 7. get original data,RLP data
// 8. status

// Set
// 1. set umulative difficulty
// 2. status

// Action
// 1. signature
// 2. check data,checking blcok data according to the document description oin 6.3.if it is digging block,the signature of sender and vsr is same.

// Init
// Parsing interface  -- RLP decode - Accepting bytes from the internet，decode block to DecodeRLP implements rlp.Decoder
type BlockStatus uint

const (
	BLockNone BlockStatus = 1 << iota //0
	BlockVerify
	BlockTmpMaxDiff
	BlockConfirm
	BlockApply
	BlockTmpMain
	BlockMain
)

type BlockType uint

const (
	BlockTypeTx      BlockType = 1
	BlockTypeMiner   BlockType = 2
	BlockTypeGenesis BlockType = 3
)

var (
	GenesisTime uint64 //1546272000000
)

// Block,digging not including signaturation，hash
//**************************
type Block interface {
	GetType() BlockType    // Get the type of block
	GetGasPrice() *big.Int // Get gas
	GetGasLimit() uint64   // Get the max value of gas
	GetRlp() []byte        // Get the original data of block
	GetHash() common.Hash  // Get block hash,including signature tx,miner block is the same
	GetDiff() *big.Int     // Get block difficulty ,pow.go,calutae sending hash(tx:including signature,miner:not including signaturatiom)

	GetCumulativeDiff() *big.Int               // Block cumulative difficulty
	SetCumulativeDiff(cumulativeDiff *big.Int) // Set cumulative difficulty

	GetTime() uint64                    // Get blcok time
	GetSender() (common.Address, error) // Get block sender,creator from signature
	GetLinks() []common.Hash            // Get block links

	GetStatus() BlockStatus       // Get status
	SetStatus(status BlockStatus) // Set status

	GetMutableInfo() *MutableInfo            // Get mutable information
	SetMutableInfo(mutableInfo *MutableInfo) // Set mutable information

	Sign(prv *ecdsa.PrivateKey) error // Signature

	Validation() error // Check data

	GetMaxLink() common.Hash        // Get the max link
	SetMaxLink(MaxLink common.Hash) // Set the max link
}

type BlockHeader struct {
	Type     BlockType // 1 tx, 2 miner
	Time     uint64    // ms  timestamp
	GasPrice *big.Int  // tls
	GasLimit uint64    // gas max value
}

type MutableInfo struct {
	Status           BlockStatus // status
	ConfirmItsNumber uint64      // confirm its time slice
	Difficulty       *big.Int    // self difficulty
	CumulativeDiff   *big.Int    // cumulative difficulty
	//MaxLink             uint8
	MaxLinkHash common.Hash
}

// BlockDecode, rlpData to block
func BlockDecode(rlpData []byte) (Block, error) {

	var ty BlockType

	if isList, lenNum := RLPList(rlpData[0]); isList && lenNum < len(rlpData) {
		if isList, lenNum1 := RLPList(rlpData[lenNum+1]); isList && lenNum+1+lenNum1 < len(rlpData) {
			ty = BlockType(rlpData[lenNum+1+lenNum1+1])
		} else {
			return nil, errors.New("rlpData is err")
		}
	} else {
		return nil, errors.New("rlpData is err")
	}

	switch ty {
	case BlockTypeTx:
		return new(TxBlock).UnRlp(rlpData)
	case BlockTypeMiner:
		return new(MinerBlock).UnRlp(rlpData)
	case BlockTypeGenesis:
		return new(GenesisBlock).UnRlp(rlpData)
	default:
		return nil, errors.New("block upRlp error")
	}
}

func RLPList(b byte) (bool, int) {
	if b < 0xF8 && b > 0xC0 {
		return true, 0
	} else if b >= 0xF8 {
		len := int(b) - 0xF7
		return true, len
	} else {
		return false, 0
	}
}

func GetMutableRlp(mutableInfo *MutableInfo) []byte {
	enc, err := rlp.EncodeToBytes(mutableInfo)
	if err != nil {
		fmt.Println("err: ", err)
	}
	return enc
}

func UnMutableRlp(mutableRLP []byte) (*MutableInfo, error) {

	newMutableInfo := new(MutableInfo)

	if err := rlp.DecodeBytes(mutableRLP, newMutableInfo); err != nil {
		return nil, err
	}

	return newMutableInfo, nil
}

func GetBlockStatus(blockStatusInfo BlockStatus) string {

	isApply := blockStatusInfo&(BlockApply|BlockConfirm) == (BlockApply | BlockConfirm)
	isReject := blockStatusInfo&BlockConfirm == BlockConfirm

	if blockStatusInfo&BlockMain == BlockMain {
		if isApply {
			return "Main|Accepted"
		} else if isReject {
			return "Main|Rejected"
		}
	} else if isApply {
		return "Accepted"
	} else if isReject {
		return "Rejected"
	}
	return "Pending"
}
