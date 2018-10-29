package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math/big"
	"sync"
	"time"
)

type MainChain struct {
	db             tosdb.Database
	stateDb        state.Database
	Tail           types.TailMainBlockInfo
	PervTail       types.TailMainBlockInfo
	MainTail       types.TailMainBlockInfo
	tailRWLock     sync.RWMutex
	mainTailRWLock sync.RWMutex
}

func (mainChain *MainChain) initTail() error {
	tailMainBlockInfo, err := storage.ReadTailMainBlockInfo(mainChain.db)
	if err != nil {
		log.Info("generate genesis")
		genesis := core.NewGenesis(mainChain.db, mainChain.stateDb, "")
		genesisBlock, err := genesis.Genesis()
		if err != nil {
			return err
		}
		mainChain.Tail = *genesisBlock
		mainChain.PervTail = *genesisBlock
		mainChain.MainTail = *genesisBlock
		return nil
	}
	mainChain.Tail = *tailMainBlockInfo
	err = mainChain.setPerv()
	return err
}

func (mainChain *MainChain) setPerv() error {
	hash := mainChain.Tail.Hash
	notSelf := false
	PervTailIsFilled := false
	Number := mainChain.Tail.Number
	count := 0
	for {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return err
		}

		if notSelf {
			Number--
			if !PervTailIsFilled {
				block := storage.ReadBlock(mainChain.db, hash)
				if block == nil {
					return fmt.Errorf("ReadBlock err")
				}
				block.SetMutableInfo(mutableInfo)

				mainChain.PervTail.Hash = hash
				mainChain.PervTail.Time = block.GetTime()
				mainChain.PervTail.Number = Number
				mainChain.PervTail.CumulativeDiff = block.GetCumulativeDiff()
				PervTailIsFilled = true
			}
		}
		if (mutableInfo.Status & types.BlockMain) != 0 {
			block := storage.ReadBlock(mainChain.db, hash)
			if block == nil {
				return fmt.Errorf("ReadBlock err")
			}
			block.SetMutableInfo(mutableInfo)

			mainChain.MainTail.Hash = hash
			mainChain.MainTail.Time = block.GetTime()
			mainChain.MainTail.Number = Number
			mainChain.MainTail.CumulativeDiff = block.GetCumulativeDiff()
			break
		}
		hash = mutableInfo.MaxLinkHash
		notSelf = true
		count++
	}
	log.Warn("setPerv loop", "count", count)
	if mainChain.MainTail.Hash == (common.Hash{}) {
		return fmt.Errorf("not found MainTail")
	}
	if mainChain.PervTail.Hash == (common.Hash{}) {
		mainChain.PervTail = mainChain.MainTail
	}
	return nil
}

func New(chainDb tosdb.Database, stateDb state.Database) (*MainChain, error) {
	var mainChain MainChain
	mainChain.db = chainDb
	mainChain.stateDb = stateDb

	err := mainChain.initTail()
	if err != nil {
		return nil, err
	}
	log.Debug("mainChain initTail finish", "Tail", mainChain.Tail, "PervTail", mainChain.PervTail, "MainTail", mainChain.MainTail)

	go func() {
		currentTime := time.Now().Unix()
		lastTime := currentTime
		for {
			currentTime = time.Now().Unix()
			if lastTime+params.TimePeriod/1000 < currentTime {
				err := mainChain.Confirm()
				if err != nil {
					log.Error(err.Error())
				}
				lastTime = currentTime
			}
			time.Sleep(time.Second)
		}
	}()

	return &mainChain, nil
}

func (mainChain *MainChain) GetTail() *types.TailMainBlockInfo {
	mainChain.tailRWLock.RLock()
	tail := mainChain.Tail
	mainChain.tailRWLock.RUnlock()
	return &tail
}

func (mainChain *MainChain) GetPervTail() (common.Hash, *big.Int) {
	mainChain.tailRWLock.RLock()
	tail := mainChain.PervTail
	mainChain.tailRWLock.RUnlock()
	return tail.Hash, tail.CumulativeDiff
}

func (mainChain *MainChain) GetMainTail() *types.TailMainBlockInfo {
	mainChain.mainTailRWLock.RLock()
	tail := mainChain.MainTail
	mainChain.mainTailRWLock.RUnlock()
	return &tail
}

func (mainChain *MainChain) UpdateTail(block types.Block) {
	mainChain.tailRWLock.RLock()
	if mainChain.Tail.CumulativeDiff.Cmp(block.GetCumulativeDiff()) < 0 {
		log.Debug("update tail", "hash", block.GetHash().String(), "block", block)
		mainChain.tailRWLock.RUnlock()
		mainChain.tailRWLock.Lock()
		if !IsTheSameTimeSlice(mainChain.Tail.Time, block.GetTime()) {
			mainChain.PervTail = mainChain.Tail
			mainChain.Tail.Number++
		}
		mainChain.Tail.Hash = block.GetHash()
		mainChain.Tail.CumulativeDiff = block.GetCumulativeDiff()
		mainChain.Tail.Time = block.GetTime()
		err := storage.WriteTailMainBlockInfo(mainChain.db, &mainChain.Tail)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debug("update tail finished", "hash", block.GetHash().String(), "mainChain.Tail", mainChain.Tail, "mainChain.PervTail", mainChain.PervTail)
		mainChain.tailRWLock.Unlock()
	} else {
		mainChain.tailRWLock.RUnlock()
	}
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

func (mainChain *MainChain) ComputeCumulativeDiff(toBeAddedBlock types.Block) (bool, error) {
	type SingleChainLinkInfo struct {
		isUpdateDiff              bool
		maxLinkHash               common.Hash
		SingleChainCumulativeDiff *big.Int
	}

	var (
		chainLinkInfo SingleChainLinkInfo
	)
	chainLinkInfo.SingleChainCumulativeDiff = big.NewInt(0)

	for _, hash := range toBeAddedBlock.GetLinks() {
		var singleChainLinkInfo SingleChainLinkInfo
		singleChainLinkInfo.SingleChainCumulativeDiff = big.NewInt(0)

		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return false, err
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return false, fmt.Errorf("ComputeCumulativeDiff ReadBlock Not found")
		}
		block.SetMutableInfo(mutableInfo)

		if (block.GetStatus() & types.BlockTmpMaxDiff) != 0 {
			if !IsTheSameTimeSlice(toBeAddedBlock.GetTime(), block.GetTime()) {
				singleChainLinkInfo.SingleChainCumulativeDiff.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff())
				singleChainLinkInfo.isUpdateDiff = true
				singleChainLinkInfo.maxLinkHash = hash
			} else {
				if block.GetDiff().Cmp(toBeAddedBlock.GetDiff()) < 0 {
					singleChainLinkInfo.SingleChainCumulativeDiff.Add(singleChainLinkInfo.SingleChainCumulativeDiff.Sub(block.GetCumulativeDiff(), block.GetDiff()), toBeAddedBlock.GetDiff())
					singleChainLinkInfo.isUpdateDiff = true
					singleChainLinkInfo.maxLinkHash = block.GetMaxLink()
				} else {
					singleChainLinkInfo.SingleChainCumulativeDiff = block.GetCumulativeDiff()
					singleChainLinkInfo.isUpdateDiff = false
					singleChainLinkInfo.maxLinkHash = hash
				}
			}
		} else {
			if !IsTheSameTimeSlice(toBeAddedBlock.GetTime(), block.GetTime()) {
				singleChainLinkInfo.SingleChainCumulativeDiff.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff())
				singleChainLinkInfo.isUpdateDiff = true
				singleChainLinkInfo.maxLinkHash = block.GetMaxLink()
			} else {
				DiffBefore := utils.CalculateWork(block.GetMaxLink())
				if DiffBefore.Cmp(toBeAddedBlock.GetDiff()) < 0 {
					singleChainLinkInfo.SingleChainCumulativeDiff.Add(singleChainLinkInfo.SingleChainCumulativeDiff.Sub(block.GetCumulativeDiff(), DiffBefore), toBeAddedBlock.GetDiff())
					mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, block.GetMaxLink())
					if err != nil {
						return false, err
					}
					singleChainLinkInfo.isUpdateDiff = true
					singleChainLinkInfo.maxLinkHash = mutableInfo.MaxLinkHash
				} else {
					singleChainLinkInfo.SingleChainCumulativeDiff = block.GetCumulativeDiff()
					singleChainLinkInfo.isUpdateDiff = false
					singleChainLinkInfo.maxLinkHash = block.GetMaxLink()
				}
			}
		}

		if chainLinkInfo.SingleChainCumulativeDiff.Cmp(singleChainLinkInfo.SingleChainCumulativeDiff) < 0 {
			chainLinkInfo = singleChainLinkInfo
		}
	}

	toBeAddedBlock.SetCumulativeDiff(chainLinkInfo.SingleChainCumulativeDiff)
	toBeAddedBlock.SetMaxLink(chainLinkInfo.maxLinkHash)
	if chainLinkInfo.isUpdateDiff {
		log.Debug("update  CumulativeDiff", "hash", toBeAddedBlock.GetHash().String())
		toBeAddedBlock.SetStatus(toBeAddedBlock.GetStatus() | types.BlockTmpMaxDiff)
	}
	return chainLinkInfo.isUpdateDiff, nil
}

//Find the main block that can confirm other blocks
func (mainChain *MainChain) findTheMainBlockThatCanConfirmOtherBlocks() ([]types.Block, uint64, uint64, error) {
	tail := mainChain.GetTail()
	number := tail.Number
	hash := tail.Hash
	now := utils.GetTimeStamp()
	var (
		canConfirm        bool
		notSelf           bool
		listBlock         []types.Block
		lastMainTimeSlice uint64
	)
	for {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("findTheMainBlockThatCanConfirmOtherBlocks ReadBlockMutableInfo Not found. hash=%s", hash.String())
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return nil, 0, 0, fmt.Errorf("findTheMainBlockThatCanConfirmOtherBlocks ReadBlock Not found. hash=%s", hash.String())
		}
		block.SetMutableInfo(mutableInfo)

		if !canConfirm && TimeSliceDifference(now, block.GetTime()) > params.ConfirmBlock {
			canConfirm = true
		}

		if notSelf {
			number--
		}

		if canConfirm {
			if (block.GetStatus() & types.BlockMain) != 0 {
				lastMainTimeSlice = utils.GetMainTime(block.GetTime())
				break
			}
			listBlock = append([]types.Block{block}, listBlock...)
			log.Debug("findTheMainBlockThatCanConfirmOtherBlocks", "hash", block.GetHash().String(), "block", block)
		}
		hash = block.GetMutableInfo().MaxLinkHash
		notSelf = true
	}
	return listBlock, lastMainTimeSlice, number + 1, nil
}

func (mainChain *MainChain) Confirm() error {
	listMainBlock, lastMainTimeSlice, blockNumber, err := mainChain.findTheMainBlockThatCanConfirmOtherBlocks()
	if err != nil {
		return err
	}
	for _, block := range listMainBlock {
		if block, err := storage.ReadMainBlock(mainChain.db, utils.GetMainTime(block.GetTime())); err == nil {
			mainChain.RollBackStatus(block.Hash)
		}
	}

	if len(listMainBlock) == 0 {
		return nil
	}

	mainBlock, err := storage.ReadMainBlock(mainChain.db, lastMainTimeSlice)
	if err != nil {
		return fmt.Errorf("%d ReadMainBlock lastMainTimeSlice fail", lastMainTimeSlice)
	}
	state, err := state.New(mainBlock.Root, mainChain.stateDb)
	if err != nil {
		return fmt.Errorf("%s state.New fail [%s]", mainBlock.Root.String(), err.Error())
	}

	for _, block := range listMainBlock {
		log.Debug("the main block confirm", "hash", block.GetHash().String(), "block", block)
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

		log.Debug("the main block confirm finished", "confirmReward.userReward", confirmReward.userReward, "confirmReward.minerReward", confirmReward.minerReward)
		if _, err := CalculatingAccounts(block, ComputeMainConfirmReward(confirmReward), state); err != nil {
			log.Info(err.Error())
		}

		if err := CalculatingMinerReward(block, blockNumber, state); err != nil {
			log.Error(err.Error())
		}

		root, err := state.Commit(false)
		if err != nil {
			return err
		}
		log.Debug("Commit finished", "root", root.String(), "hash", block.GetHash().String())
		err = state.Database().TrieDB().Commit(root, true)
		if err != nil {
			return err
		}
		err = storage.WriteMainBlock(mainChain.db, &types.MainBlockInfo{Hash: block.GetHash(), Root: root}, mainTimeSlice)
		if err != nil {
			return err
		}
		if block == listMainBlock[len(listMainBlock)-1] {
			mainChain.mainTailRWLock.Lock()
			log.Debug("begin update MainTail", "MainTail", mainChain.MainTail, "block", block)
			mainChain.MainTail.Hash = block.GetHash()
			mainChain.MainTail.Time = block.GetTime()
			mainChain.MainTail.Number = blockNumber
			mainChain.MainTail.CumulativeDiff = block.GetCumulativeDiff()
			log.Debug("update MainTail finished", "MainTail", mainChain.MainTail)
			mainChain.mainTailRWLock.Unlock()
		}
		blockNumber++
	}
	return nil
}

func (mainChain *MainChain) RollBackStatus(hash common.Hash) error {
	log.Debug("RollBackStatus main block", "hash", hash)
	block := storage.ReadBlock(mainChain.db, hash)
	if block == nil {
		return fmt.Errorf("RollBackStatus ReadBlock Not found")
	}
	mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
	if err != nil {
		return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
	}
	log.Debug("RollBackStatus main block", "block", block, "mutableInfo", mutableInfo)

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
	log.Debug("singleRollBackStatus", "hash", block.GetHash().String(), "block", block)
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
	log.Debug("begin singleBlockConfirm", "block", block.GetHash())
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
			log.Error(err.Error())
		}
		log.Debug("singleBlockConfirm ancestors finished", "hash", block.GetHash().String(), "linksReward.userReward", linksReward.userReward, "linksReward.minerReward", linksReward.minerReward)
		if linksReward != nil {
			confirmRewardInfo.userReward.Add(confirmRewardInfo.userReward, linksReward.userReward)
			confirmRewardInfo.minerReward.Add(confirmRewardInfo.minerReward, linksReward.minerReward)
		}
	}
	log.Debug("singleBlockConfirm self", "hash", block.GetHash().String(),
		"block", block, "confirmRewardInfo.userReward", confirmRewardInfo.userReward.String(), "confirmRewardInfo.minerReward", confirmRewardInfo.minerReward.String())
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
