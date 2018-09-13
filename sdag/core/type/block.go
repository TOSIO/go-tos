package _type

import (
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/crypto/sha3"
	"github.com/TOSIO/go-tos/devbase/rlp"
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

type block interface {
	Hash() common.Hash      //获取区块hash
	Diff() *big.Int         //获取区块难度
	Time() *big.Int         //获取区块时间
	Sender() common.Address //获取区块发送者，即创建者
	Sign()
	RlpEncode()
	RlpDecode()
	Validation()
	//签名
	//RLP编解码
}

type BlockHeader struct {
	Type uint32
	//Time       uint64
	GasPrice   *big.Int
	GasLimit   uint64
	Difficulty *big.Int //`json:"difficulty"       gencodec:"required"`
	Time       *big.Int
	//Hash     common.Hash
}

//获取区块hash并用RLP编码
func (b *BlockHeader) Hash() common.Hash {
	return RlpEncode(b)
}

//把hash用RLP编码
func RlpEncode(h interface{}) (z common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, h)
	hw.Sum(z[:0])
	return z
}

type Block struct {
	header *BlockHeader
}

//获取区块时间
func (b *Block) Time() *big.Int {
	return new(big.Int).Set(b.header.Time)
}
