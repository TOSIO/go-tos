package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
)

var (
	db tosdb.Database
)

func SetDB(chainDb tosdb.Database) {
	db = chainDb
}

type Mainchain struct {
}

func New() (*Mainchain, error) {
	return &Mainchain{}, nil
}

func (mc *Mainchain) GetLastTempMainBlkSlice() uint64 {
	return 0
}

func IsTheSameTimeSlice(b1, b2 types.Block) bool {
	return utils.GetMainTime(b1.GetTime()) == utils.GetMainTime(b2.GetTime())
}

func SetCumulativeDiff(toBeAddedBlock types.Block) error {
	CumulativeDiff := big.NewInt(0)
	var index uint8
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
			if !IsTheSameTimeSlice(toBeAddedBlock, block) {
				SingleChainCumulativeDiff = SingleChainCumulativeDiff.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff())
				break
			}

			if mutableInfo.Status&types.BlockTmpMaxDiff != 0 {
				SingleChainCumulativeDiff = SingleChainCumulativeDiff.Add(SingleChainCumulativeDiff.Sub(block.GetCumulativeDiff(), block.GetDiff()), toBeAddedBlock.GetDiff())
				break
			}
			hash = block.GetLinks()[mutableInfo.MaxLink]
		}
		if CumulativeDiff.Cmp(SingleChainCumulativeDiff) < 0 {
			CumulativeDiff = SingleChainCumulativeDiff
			index = uint8(i)
		}
	}

	toBeAddedBlock.SetCumulativeDiff(CumulativeDiff)
	toBeAddedBlock.SetMaxLinks(index)
	return nil
}
