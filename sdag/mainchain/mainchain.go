package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
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
			//Tail.Number++
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

func TimeSliceDifference(t1, t2 uint64) uint64 {
	return utils.GetMainTime(t1) - utils.GetMainTime(t2)
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
			if (mutableInfo.Status & types.BlockTmpMaxDiff) != 0 {
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

//Find the main block that can confirm other blocks
func findTheMainBlockThatCanConfirmOtherBlocks() ([]types.Block, error) {
	now := utils.GetTimeStamp()
	hash := Tail.Hash
	var canConfirm bool
	var currentTimeSliceAdded bool
	var currentTimeSlice uint64
	var listBlock []types.Block
	for {
		block := storage.ReadBlock(db, hash)
		if block == nil {
			return nil, fmt.Errorf("ReadBlock Not found")
		}

		if !canConfirm && TimeSliceDifference(now, block.GetTime()) > params.ConfirmBlock {
			canConfirm = true
		}

		if canConfirm {
			if currentTimeSlice != utils.GetMainTime(block.GetTime()) {
				currentTimeSliceAdded = false
				currentTimeSlice = utils.GetMainTime(block.GetTime())
			}
			if !currentTimeSliceAdded {
				mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
				if err != nil {
					return nil, err
				}
				if (mutableInfo.Status & types.BlockMain) != 0 {
					break
				}
				if (mutableInfo.Status & types.BlockTmpMaxDiff) != 0 {
					block.SetMutableInfo(mutableInfo)
					listBlock = append(listBlock, block)
					currentTimeSliceAdded = true
				}
			}
		}
		hash = block.GetLinks()[block.GetMutableInfo().MaxLink]
	}
	return listBlock, nil
}
func Confirm() error {
	listMainBlock, err := findTheMainBlockThatCanConfirmOtherBlocks()
	if err != nil {
		return err
	}
	for _, block := range listMainBlock {
		if block, err := storage.ReadMainBlock(db, utils.GetMainTime(block.GetTime())); err != nil {
			RollBackStatus(block.Hash)
		}
	}

	for _, block := range listMainBlock {
		if err := singleBlockConfirm(block); err != nil {
			return err
		}
		info := block.GetMutableInfo()
		info.Status |= types.BlockMain
		if err := storage.WriteBlockMutableInfo(db, block.GetHash(), info); err != nil {
			return nil
		}
	}
	return nil
}

func RollBackStatus(hash common.Hash) error {
	block := storage.ReadBlock(db, hash)
	if block == nil {
		return fmt.Errorf("ReadBlock Not found")
	}
	mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
	if err != nil {
		return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
	}
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsTimeSlice != block.GetTime() {
			continue
		}
		block := storage.ReadBlock(db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = singleRollBackStatus(block); err != nil {
			return err
		}
	}
	mutableInfo.Status &= ^types.BlockMain
	storage.WriteBlockMutableInfo(db, block.GetHash(), mutableInfo)
	return nil
}

func singleRollBackStatus(block types.Block) error {
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsTimeSlice != block.GetTime() {
			continue
		}
		block := storage.ReadBlock(db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = singleRollBackStatus(block); err != nil {
			return err
		}
	}
	mutableInfo := block.GetMutableInfo()
	mutableInfo.Status &= ^types.BlockConfirm
	mutableInfo.ConfirmItsTimeSlice = 0
	storage.WriteBlockMutableInfo(db, block.GetHash(), mutableInfo)
	return nil
}

func singleBlockConfirm(block types.Block) error {
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if (mutableInfo.Status & types.BlockConfirm) != 0 {
			continue
		}
		block := storage.ReadBlock(db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = singleBlockConfirm(block); err != nil {
			return err
		}
	}

	return nil
}
