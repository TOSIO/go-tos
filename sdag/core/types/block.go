package types
import (
	"math/big"


	"github.com/TOSIO/go-tos/devbase/common"
_	"github.com/TOSIO/go-tos/devbase/rlp"
)

//定义block公共的接口
//1. 获取区块hash
//2. 区块难度   挖矿块与交易块略有不同，参考6.1 区块难度
//3. 区块时间
//4. 区块发送者，根据vrs签名获取发送者地址
//5. 区块签名
//6. RLP编码 - 带签名或不带签名
//7. RLP解码 - 从网络收到字节码，解码成block
//8. 验证 - 根据文档6.3描述检查区块数据  1，2， 3，  如果是挖矿区块 sender与vsr中解析中的签名是相同的


//挖矿不包括签名，hash
//**************************
type Block interface {
	Hash() common.Hash 		//获取区块hash,包括签名,tx,miner block is the same
	Diff(hash common.Hash) *big.Int	  		//获取区块难度,pow.go,calutae 传入hash(tx:包含签名,miner:不包括签名 )
	Time() uint64				//获取区块时间
	Sender() common.Address     //获取区块发送者，即创建者,从签名获取
	Sign() 
	Links() []common.Address
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


