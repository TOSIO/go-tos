package types
import (
	"math/big"
	"github.com/TOSIO/go-tos/devbase/common"
	"errors"
	"crypto/ecdsa"
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

const(
	BLockNone BlockStatus = 1 << iota //0
	BlockVerify
	BlockConfirm
	BlockApply
	BlockTmpMain
	BlockMain
)


//挖矿不包括签名，hash
//**************************
type Block interface {
	GetRlp() []byte 		   //获取区块原始数据
	GetHash() common.Hash 		   //获取区块hash,包括签名,tx,miner block is the same
	GetDiff() *big.Int	  		   //获取区块难度,pow.go,calutae 传入hash(tx:包含签名,miner:不包括签名 )

	GetCumulativeDiff() *big.Int  //区块累积难度
	SetCumulativeDiff(cumulativeDiff *big.Int) //设置累积难度

	GetTime() int64			   //获取区块时间
	GetSender() (common.Address, error)    //获取区块发送者，即创建者,从签名获取
	GetLinks() []common.Hash   //获取区块链接组

	GetStatus() BlockStatus       //获取状态
	SetStatus(status BlockStatus) //设置状态

	Sign(prv *ecdsa.PrivateKey)  error //签名

	//Validation() error   // (check data,校验解签名)
}

type BlockHeader struct {
	Type int32  //1 tx, 2 miner
	Time int64 //ms  timestamp
	GasPrice *big.Int  //tls
	GasLimit uint64   //gas max value
}





