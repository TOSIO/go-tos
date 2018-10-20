package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
	"time"
)

var (
	emptyChan = make(chan struct{}, 1)
)

type MainChain struct {
	db       tosdb.Database
	stateDb  state.Database
	Tail     TailBlock
	pervTail TailBlock
}

type TailBlock struct {
	Hash           common.Hash
	CumulativeDiff *big.Int
	Number         uint64
	Time           uint64
}

func (mainChain *MainChain) GetPervTail() (common.Hash, *big.Int) {
	emptyChan <- struct{}{}
	tail := mainChain.pervTail
	<-emptyChan
	return tail.Hash, tail.CumulativeDiff
}

func New(chainDb tosdb.Database, Db state.Database) (*MainChain, error) {
	var mainChain MainChain
	mainChain.db = chainDb
	mainChain.stateDb = Db
	go func() {
		currentTime := time.Now().Unix()
		lastTime := currentTime
		for {
			currentTime = time.Now().Unix()
			if lastTime+params.TimePeriod/1000 < currentTime {
				mainChain.Confirm()
				lastTime = currentTime
			}
			time.Sleep(time.Second)
		}
	}()

	return &mainChain, nil
}

func (mainChain *MainChain) UpdateTail(block types.Block) {
	if mainChain.Tail.CumulativeDiff.Cmp(block.GetCumulativeDiff()) < 0 {
		emptyChan <- struct{}{}
		if !IsTheSameTimeSlice(mainChain.Tail.Time, block.GetTime()) {
			mainChain.pervTail = mainChain.Tail
			mainChain.Tail.Number++
		}
		mainChain.Tail.Hash = block.GetHash()
		mainChain.Tail.CumulativeDiff = block.GetCumulativeDiff()
		<-emptyChan
		//todo: wirte DB
	}
}

func (mainChain *MainChain) GetTail() *TailBlock {
	emptyChan <- struct{}{}
	tail := mainChain.Tail
	<-emptyChan
	return &tail
}

func (mainChain *MainChain) GetLastTempMainBlkSlice() uint64 {
	return utils.GetMainTime(mainChain.GetTail().Time)
}

func IsTheSameTimeSlice(t1, t2 uint64) bool {
	return utils.GetMainTime(t1) == utils.GetMainTime(t2)
}

func TimeSliceDifference(t1, t2 uint64) uint64 {
	return utils.GetMainTime(t1) - utils.GetMainTime(t2)
}

func (mainChain *MainChain) ComputeCumulativeDiff(toBeAddedBlock types.Block) error {
	CumulativeDiff := big.NewInt(0)
	var index int
	linksIsUpdateDiff := make(map[int]bool)
	for i, hash := range toBeAddedBlock.GetLinks() {
		SingleChainCumulativeDiff := big.NewInt(0)
		for {
			mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
			if err != nil {
				return err
			}
			block := storage.ReadBlock(mainChain.db, hash)
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
		mainChain.UpdateTail(toBeAddedBlock)
		//if err := storage.WriteBlockMutableInfo(mainChain.db, toBeAddedBlock.GetHash(), toBeAddedBlock.GetMutableInfo()); err != nil {
		//	return err
		//}
	}
	return nil
}

//Find the main block that can confirm other blocks
func (mainChain *MainChain) findTheMainBlockThatCanConfirmOtherBlocks() ([]types.Block, uint64, uint64, error) {
	tail := mainChain.GetTail()
	number := tail.Number
	hash := tail.Hash
	now := utils.GetTimeStamp()
	var canConfirm bool
	var currentTimeSliceAdded bool
	var currentTimeSlice uint64
	var listBlock []types.Block
	var lastMainTimeSlice uint64
	for {
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return nil, 0, 0, fmt.Errorf("ReadBlock Not found")
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
				mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
				if err != nil {
					return nil, 0, 0, err
				}
				if (mutableInfo.Status & types.BlockMain) != 0 {
					lastMainTimeSlice = currentTimeSlice
					break
				}
				if (mutableInfo.Status & types.BlockTmpMaxDiff) != 0 {
					block.SetMutableInfo(mutableInfo)
					listBlock = append(listBlock, block)
					currentTimeSliceAdded = true
					number--
				}
			}
		}
		hash = block.GetLinks()[block.GetMutableInfo().MaxLink]
	}
	return listBlock, lastMainTimeSlice, number, nil
}

func (mainChain *MainChain) Confirm() error {
	listMainBlock, lastMainTimeSlice, blockNumber, err := mainChain.findTheMainBlockThatCanConfirmOtherBlocks()
	if err != nil {
		return err
	}
	for _, block := range listMainBlock {
		if block, err := storage.ReadMainBlock(mainChain.db, utils.GetMainTime(block.GetTime())); err != nil {
			mainChain.RollBackStatus(block.Hash)
		}
	}

	mainBlock, err := storage.ReadMainBlock(mainChain.db, lastMainTimeSlice)
	if err == nil {
		return fmt.Errorf("%d ReadMainBlock lastMainTimeSlice fail", lastMainTimeSlice)
	}
	state, err := state.New(mainBlock.Root, mainChain.stateDb)
	if err != nil {
		return fmt.Errorf("%s state.New fail [%s]", mainBlock.Root.String(), err.Error())
	}

	for _, block := range listMainBlock {
		mainTimeSlice := utils.GetMainTime(block.GetTime())
		confirmReward, err := mainChain.singleBlockConfirm(block, mainTimeSlice, state)
		if err != nil {
			return err
		}
		info := block.GetMutableInfo()
		info.Status |= types.BlockMain
		block.SetMutableInfo(info)
		if err := storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), info); err != nil {
			return err
		}

		CalculatingAccounts(block, ComputeMainConfirmReward(confirmReward), state)

		CalculatingMinerReward(block, blockNumber, state)

		root, err := state.Commit(false)
		if err != nil {
			return err
		}
		err = state.Database().TrieDB().Commit(root, true)
		if err != nil {
			return err
		}
		err = storage.WriteMainBlock(mainChain.db, &types.MainBlock{Hash: block.GetHash(), Root: root}, mainTimeSlice)
		if err != nil {
			return err
		}
		blockNumber++
	}
	return nil
}

func (mainChain *MainChain) RollBackStatus(hash common.Hash) error {
	block := storage.ReadBlock(mainChain.db, hash)
	if block == nil {
		return fmt.Errorf("ReadBlock Not found")
	}
	mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
	if err != nil {
		return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
	}
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsTimeSlice != block.GetTime() {
			continue
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = mainChain.singleRollBackStatus(block); err != nil {
			return err
		}
	}
	mutableInfo.Status &= ^types.BlockMain
	storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), mutableInfo)
	return nil
}

func (mainChain *MainChain) singleRollBackStatus(block types.Block) error {
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsTimeSlice != block.GetTime() {
			continue
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = mainChain.singleRollBackStatus(block); err != nil {
			return err
		}
	}
	mutableInfo := block.GetMutableInfo()
	mutableInfo.Status &= ^types.BlockConfirm
	mutableInfo.ConfirmItsTimeSlice = 0
	storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), mutableInfo)
	return nil
}

func (mainChain *MainChain) singleBlockConfirm(block types.Block, MainTimeSlice uint64, state *state.StateDB) (*ConfirmRewardInfo, error) {
	var (
		transactionSuccess bool
		confirmRewardInfo  ConfirmRewardInfo
	)
	confirmRewardInfo.Init()
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return nil, fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if (mutableInfo.Status & (types.BlockConfirm | types.BlockMain)) != 0 {
			continue
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return nil, fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		linksReward, err := mainChain.singleBlockConfirm(block, MainTimeSlice, state)
		if err != nil {
		}
		if linksReward != nil {
			confirmRewardInfo.userReward.Add(confirmRewardInfo.userReward, linksReward.userReward)
			confirmRewardInfo.minerReward.Add(confirmRewardInfo.minerReward, linksReward.minerReward)
		}
	}
	var err error
	confirmRewardInfo.userReward, err = CalculatingAccounts(block, confirmRewardInfo.userReward, state)
	if err == nil {
		transactionSuccess = true
	}
	info := block.GetMutableInfo()
	info.Status |= types.BlockConfirm
	if transactionSuccess {
		info.Status |= types.BlockApply
	}
	info.ConfirmItsTimeSlice = MainTimeSlice
	if err = storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), info); err != nil {
		log.Error(err.Error())
	}

	return ComputeConfirmReward(&confirmRewardInfo), err
}
