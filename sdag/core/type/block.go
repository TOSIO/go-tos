package _type

import (
	"math/big"


	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/rlp"
)

//定义block公共的接口
//1. 获取区块hash
//2. 区块难度
//3. 区块时间
//4. 区块发送者，根据vrs签名获取发送者地址
//5. 区块签名
//6. RLP编码
//7. RLP解码
//8. verfiy  根据文档6.3描述检查区块数据



type block interface {
	Hash() common.hash 		//获取区块hash
	Diff() *big.Int	  		//获取区块难度
	Time() uint64				//获取区块时间
	Sender() common.Address     //获取区块发送者，即创建者

	//签名
	//RLP编解码
}

type BlockHeader struct {
	Version uint32
	Type uint32
	Time uint64
	GasPrice *big.Int
	GasLimit uint64
}






