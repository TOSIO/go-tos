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

//定义block公共的接口

// get接口
//1. 获取区块hash
//2. 区块难度   挖矿块与交易块略有不同，参考6.1 区块难度
//3. 累积难度
//4. 区块时间
//5. 区块发送者 根据vrs签名获取发送者地址
//6. 区块链接组
//7. 获取原始数据 整体RLP数据
//8. status

//set
//1. 设置累积难度
//2. status

//动作
//1. 签名
//2. 校验数据 - 根据文档6.3描述检查区块数据  1，2， 3，  如果是挖矿区块 sender与vsr中解析中的签名是相同的

//初始化
//解析接口 -- RLP解码 - 从网络收到字节码，解码成block  DecodeRLP implements rlp.Decoder
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

//挖矿不包括签名，hash
//**************************
type Block interface {
	GetType() BlockType    //获取区块类型
	GetGasPrice() *big.Int //获取gas
	GetGasLimit() uint64   //获取gas max value
	GetRlp() []byte        //获取区块原始数据
	GetHash() common.Hash  //获取区块hash,包括签名,tx,miner block is the same
	GetDiff() *big.Int     //获取区块难度,pow.go,calutae 传入hash(tx:包含签名,miner:不包括签名 )

	GetCumulativeDiff() *big.Int               //区块累积难度
	SetCumulativeDiff(cumulativeDiff *big.Int) //设置累积难度

	GetTime() uint64                    //获取区块时间
	GetSender() (common.Address, error) //获取区块发送者，即创建者,从签名获取
	GetLinks() []common.Hash            //获取区块链接组

	GetStatus() BlockStatus       //获取状态
	SetStatus(status BlockStatus) //设置状态

	GetMutableInfo() *MutableInfo            //获取易变信息
	SetMutableInfo(mutableInfo *MutableInfo) //设置易变信息

	Sign(prv *ecdsa.PrivateKey) error //签名

	Validation() error // (check data,校验解签名)

	GetMaxLink() common.Hash        //获取最大连接
	SetMaxLink(MaxLink common.Hash) //设置最大连接
}

type BlockHeader struct {
	Type     BlockType //1 tx, 2 miner
	Time     uint64    //ms  timestamp
	GasPrice *big.Int  //tls
	GasLimit uint64    //gas max value
}

type MutableInfo struct {
	Status              BlockStatus //status
	ConfirmItsTimeSlice uint64      //Confirm its time slice
	Difficulty          *big.Int    //self difficulty
	CumulativeDiff      *big.Int    //cumulative difficulty
	//MaxLink             uint8
	MaxLinkHash common.Hash
}

//数据解析
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

	if ty == BlockTypeTx {
		return new(TxBlock).UnRlp(rlpData)
	} else if ty == BlockTypeMiner {
		return new(MinerBlock).UnRlp(rlpData)
	} else if ty == BlockTypeGenesis {
		return new(GenesisBlock).UnRlp(rlpData)
	}

	return nil, errors.New("block upRlp error")
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
