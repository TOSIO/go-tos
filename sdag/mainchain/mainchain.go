package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

var (
	db       tosdb.Database
	Tail     TailBlock
	pervTail TailBlock
)

type TailBlock struct {
	Hash           common.Hash
	CumulativeDiff *big.Int
	Number         uint64
	Time           uint64
}

func SetDB(chainDb tosdb.Database) {
	db = chainDb
}

type Mainchain struct {
}

func UpdateTail(block types.Block) {
	if Tail.CumulativeDiff.Cmp(block.GetCumulativeDiff()) < 0 {
		if !IsTheSameTimeSlice(Tail.Time, block.GetTime()) {
			pervTail = Tail
			Tail.Number++
		}
		Tail.Hash = block.GetHash()
		Tail.CumulativeDiff = block.GetCumulativeDiff()
		//todo: wirte DB
	}
}

func New() (*Mainchain, error) {
	return &Mainchain{}, nil
}

func (mc *Mainchain) GetLastTempMainBlkSlice() uint64 {
	return 0
}

func IsTheSameTimeSlice(t1, t2 uint64) bool {
	return utils.GetMainTime(t1) == utils.GetMainTime(t2)
}

func ComputeCumulativeDiff(toBeAddedBlock types.Block) error {
	CumulativeDiff := big.NewInt(0)
	var index int
	linksIsUpdateDiff := make(map[int]bool)
	for i, hash := range toBeAddedBlock.GetLinks() {
		SingleChainCumulativeDiff := big.NewInt(0)
		for {
			mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
			if err != nil {
				return err
			}
			block := storage.ReadBlock(db, hash)
			if block == nil {
				return fmt.Errorf("ReadBlock Not found")
			}

			if !IsTheSameTimeSlice(toBeAddedBlock.GetTime(), block.GetTime()) {
				SingleChainCumulativeDiff = SingleChainCumulativeDiff.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff())
				linksIsUpdateDiff[i] = true
				break
			}
			if mutableInfo.Status&types.BlockTmpMaxDiff != 0 {
				if block.GetDiff().Cmp(toBeAddedBlock.GetDiff()) < 0 {
					SingleChainCumulativeDiff = SingleChainCumulativeDiff.Add(SingleChainCumulativeDiff.Sub(block.GetCumulativeDiff(), block.GetDiff()), toBeAddedBlock.GetDiff())
					linksIsUpdateDiff[i] = true
				} else {
					SingleChainCumulativeDiff = block.GetCumulativeDiff()
				}
				break
			}
			hash = block.GetLinks()[mutableInfo.MaxLink]
		}
		if CumulativeDiff.Cmp(SingleChainCumulativeDiff) < 0 {
			CumulativeDiff = SingleChainCumulativeDiff
			index = i
		}
	}

	toBeAddedBlock.SetCumulativeDiff(CumulativeDiff)
	toBeAddedBlock.SetMaxLinks(uint8(index))
	if linksIsUpdateDiff[index] {
		toBeAddedBlock.SetStatus(toBeAddedBlock.GetStatus() | types.BlockTmpMaxDiff)
		UpdateTail(toBeAddedBlock)
	}
	return nil
}
