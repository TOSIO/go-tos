package types
import (
	"math/big"


	"github.com/TOSIO/go-tos/devbase/common"
_	"github.com/TOSIO/go-tos/devbase/rlp"
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
//解析接口 -- RLP解码 - 从网络收到字节码，解码成block
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
	Hash() common.Hash 		   //获取区块hash,包括签名,tx,miner block is the same
	Diff() *big.Int	  		   //获取区块难度,pow.go,calutae 传入hash(tx:包含签名,miner:不包括签名 )
	CumulativeDiff() *big.Int  //区块累积难度
	Time() uint64			   //获取区块时间
	Sender() common.Address    //获取区块发送者，即创建者,从签名获取
	Links() []common.Address
	RlpString() []byte 		  //获取区块原始数据
	Status() BlockStatus   //获取状态

	SetCumulativeDiff(diff *big.Int)
	SetStatus(status BlockStatus) //设置状态

	Sign()
	
	RlpEncode()
	RlpDecode()
	Validation()// (check data,校验解签名)
	//签名
	//RLP编解码
}

type BlockHeader struct {
	Type uint32
	Time *big.Int
	GasPrice *big.Int
	GasLimit uint64
}



