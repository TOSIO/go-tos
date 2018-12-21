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
	"github.com/TOSIO/go-tos/sdag/core/vm"
	"github.com/TOSIO/go-tos/services/messagequeue"
	"math/big"
	"strconv"
	"sync"
	"time"
)

type LocalBlockNotice struct {
	Hash    common.Hash
	Address common.Address
	Action  int
}

const (
	noneAction = iota //0
	ConfirmAction
	RollBackAction
)

type MainChain struct {
	db                 tosdb.Database
	stateDb            state.Database
	Genesis            *core.Genesis
	Tail               types.TailMainBlockInfo
	PervTail           types.TailMainBlockInfo
	MainTail           types.TailMainBlockInfo
	tailRWLock         sync.RWMutex
	mainTailRWLock     sync.RWMutex
	ChainConfig        *params.ChainConfig
	VMConfig           vm.Config
	CurrentMainBlock   types.Block
	LocalBlockNotice   chan LocalBlockNotice
	LocalAddress       map[common.Address]bool
	LocalAddressRWLock sync.RWMutex
	networkId          uint64
	mq                 *messagequeue.MessageQueue
	lastState          *state.StateDB
}

func (mainChain *MainChain) AddLocalAddress(address common.Address) {
	mainChain.LocalAddressRWLock.RLock()
	if !mainChain.LocalAddress[address] {
		mainChain.LocalAddressRWLock.RUnlock()
		mainChain.LocalAddressRWLock.Lock()
		mainChain.LocalAddress[address] = true
		mainChain.LocalAddressRWLock.Unlock()
	} else {
		mainChain.LocalAddressRWLock.RUnlock()
	}
}

func (mainChain *MainChain) LocalBlockNoticeSend(hash common.Hash, address common.Address, action int) {
	mainChain.LocalAddressRWLock.RLock()
	if mainChain.LocalAddress[address] {
		mainChain.LocalAddressRWLock.RUnlock()
		mainChain.LocalBlockNotice <- LocalBlockNotice{hash, address, action}
	} else {
		mainChain.LocalAddressRWLock.RUnlock()
	}
}
func (mainChain *MainChain) LocalBlockNoticeChan() chan LocalBlockNotice {
	return mainChain.LocalBlockNotice
}

func (mainChain *MainChain) initTail() error {
	var err error
	mainChain.Genesis, err = core.NewGenesis(mainChain.db, mainChain.stateDb, "", mainChain.networkId)
	if err != nil {
		return err
	}
	tailMainBlockInfo, err := storage.ReadTailBlockInfo(mainChain.db)
	if err != nil {
		log.Info("generate genesis")
		genesisBlock, err := mainChain.Genesis.Genesis()
		if err != nil {
			return err
		}
		mainChain.Tail = *genesisBlock
		mainChain.PervTail = *genesisBlock
		mainChain.MainTail = *genesisBlock
		return nil
	}
	mainChain.Genesis.GetGenesisHash()
	mainChain.Tail = *tailMainBlockInfo
	tailMainBlockInfo, err = storage.ReadTailMainBlockInfo(mainChain.db)
	if err != nil {
		return err
	}
	mainChain.MainTail = *tailMainBlockInfo
	err = mainChain.setPerv()
	return err
}

func (mainChain *MainChain) setPerv() error {
	hash := mainChain.Tail.Hash

	mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
	if err != nil {
		return err
	}

	if (mutableInfo.Status & types.BlockMain) != 0 {
		mainChain.PervTail = mainChain.Tail
		return nil
	}
	hash = mutableInfo.MaxLinkHash
	mutableInfo, err = storage.ReadBlockMutableInfo(mainChain.db, hash)
	if err != nil {
		return err
	}

	block := storage.ReadBlock(mainChain.db, hash)
	if block == nil {
		return fmt.Errorf("ReadBlock err")
	}
	block.SetMutableInfo(mutableInfo)
	mainChain.PervTail.Hash = block.GetHash()
	mainChain.PervTail.Time = block.GetTime()
	mainChain.PervTail.CumulativeDiff = block.GetCumulativeDiff()
	return nil
}

func New(chainDb tosdb.Database, stateDb state.Database, VMConfig vm.Config, networkId uint64, mq *messagequeue.MessageQueue) (*MainChain, error) {
	var mainChain MainChain
	mainChain.db = chainDb
	mainChain.stateDb = stateDb
	mainChain.VMConfig = VMConfig
	mainChain.LocalBlockNotice = make(chan LocalBlockNotice)
	mainChain.LocalAddress = make(map[common.Address]bool)
	mainChain.networkId = networkId
	mainChain.mq = mq

	err := mainChain.initTail()
	if err != nil {
		return nil, err
	}
	log.Debug("mainChain initTail finish", "Tail", mainChain.Tail.String(), "PervTail", mainChain.PervTail.String(), "MainTail", mainChain.MainTail.String())

	mainBlock, err := storage.ReadMainBlock(mainChain.db, mainChain.MainTail.Number)
	if err != nil {
		return nil, fmt.Errorf("initTail ReadMainBlock lastMainNunber:%d fail", mainChain.MainTail.Number)
	}
	state, err := state.New(mainBlock.Root, mainChain.stateDb)
	if err != nil {
		return nil, fmt.Errorf("%s state.New fail [%s]", mainBlock.Root.String(), err.Error())
	}
	mainChain.lastState = state

	genesisHash, err := mainChain.Genesis.GetGenesisHash()
	if err != nil {
		return nil, err
	}
	if genesisHash == params.MainnetGenesisHash {
		mainChain.ChainConfig = params.MainnetChainConfig
	} else {
		mainChain.ChainConfig = params.TestnetChainConfig
	}

	go func() {
		currentTime := time.Now().Unix()
		lastTime := currentTime
		for {
			currentTime = time.Now().Unix()
			if lastTime+params.TimePeriod/1000 < currentTime {
				err := mainChain.Confirm()
				if err != nil {
					log.Error("confirm error:" + err.Error())
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

func (mainChain *MainChain) GetLastState() *state.StateDB {
	var state *state.StateDB
	mainChain.mainTailRWLock.RLock()
	state = mainChain.lastState
	mainChain.mainTailRWLock.RUnlock()
	return state
}

func (mainChain *MainChain) GetGenesisHash() (common.Hash, error) {
	return mainChain.Genesis.GetGenesisHash()
}

func (mainChain *MainChain) GetNextMain(hash common.Hash) (common.Hash, *types.MutableInfo, error) {
	tail := mainChain.GetTail()
	var (
		err         error
		mutableInfo *types.MutableInfo
		returnHash  = tail.Hash
		count       int64
	)

	for {
		mutableInfo, err = storage.ReadBlockMutableInfo(mainChain.db, returnHash)
		count++
		if err != nil {
			log.Info("GetNextMain loop", "count", count)
			return common.Hash{}, nil, err
		}
		if mutableInfo.MaxLinkHash == hash {
			log.Info("GetNextMain loop", "count", count)
			return returnHash, mutableInfo, nil
		}
		returnHash = mutableInfo.MaxLinkHash
	}
	log.Info("GetNextMain not found hash, loop", "count", count)
	return common.Hash{}, nil, fmt.Errorf("not found hash")
}

func (mainChain *MainChain) UpdateTail(block types.Block) {
	mainChain.tailRWLock.RLock()
	if mainChain.Tail.CumulativeDiff.Cmp(block.GetCumulativeDiff()) < 0 {
		log.Debug("update tail\n" + block.String())
		mainChain.tailRWLock.RUnlock()
		mainChain.tailRWLock.Lock()
		if !IsTheSameTimeSlice(mainChain.Tail.Time, block.GetTime()) {
			mainChain.PervTail = mainChain.Tail
		}
		mainChain.Tail.Hash = block.GetHash()
		mainChain.Tail.CumulativeDiff = block.GetCumulativeDiff()
		mainChain.Tail.Time = block.GetTime()
		mainChain.Tail.Number = 0
		err := storage.WriteTailBlockInfo(mainChain.db, &mainChain.Tail)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debug("update tail finished", "hash", block.GetHash().String(), "mainChain.Tail", mainChain.Tail.String(), "mainChain.PervTail", mainChain.PervTail.String())
		mainChain.tailRWLock.Unlock()
	} else {
		mainChain.tailRWLock.RUnlock()
	}
}

func (mainChain *MainChain) GetLastTempMainBlkSlice() uint64 {
	return utils.GetMainTime(mainChain.GetTail().Time)
}

func (mainChain *MainChain) GetMainBlock(number uint64) types.Block {
	mainBlock, err := storage.ReadMainBlock(mainChain.db, number)
	if err != nil {
		log.Error("ReadMainBlock err:" + err.Error())
		return nil
	}
	block := storage.ReadBlock(mainChain.db, mainBlock.Hash)
	if block == nil {
		log.Error("ReadBlock err:" + err.Error())
		return nil
	}
	MutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, mainBlock.Hash)
	if err != nil {
		log.Error("ReadBlockMutableInfo err:" + err.Error())
		return nil
	}
	block.SetMutableInfo(MutableInfo)
	return block
}

func IsTheSameTimeSlice(t1, t2 uint64) bool {
	return utils.GetMainTime(t1) == utils.GetMainTime(t2)
}

func TimeSliceDifference(t1, t2 uint64) uint64 {
	return utils.GetMainTime(t1) - utils.GetMainTime(t2)
}

type SingleChainLinkInfo struct {
	isUpdateDiff              bool
	maxLinkHash               common.Hash
	SingleChainCumulativeDiff *big.Int
}

func (singleChainLinkInfo *SingleChainLinkInfo) set(isUpdateDiff bool, maxLinkHash common.Hash, SingleChainCumulativeDiff *big.Int) {
	singleChainLinkInfo.isUpdateDiff = isUpdateDiff
	singleChainLinkInfo.maxLinkHash = maxLinkHash
	singleChainLinkInfo.SingleChainCumulativeDiff = SingleChainCumulativeDiff
}

func (mainChain *MainChain) ComputeCumulativeDiff(toBeAddedBlock types.Block) (bool, error) {
	var (
		chainLinkInfo SingleChainLinkInfo
	)
	chainLinkInfo.SingleChainCumulativeDiff = big.NewInt(0)

	for _, hash := range toBeAddedBlock.GetLinks() {
		var singleChainLinkInfo SingleChainLinkInfo
		big0 := big.NewInt(0)

		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return false, fmt.Errorf(hash.String() + " " + err.Error())
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return false, fmt.Errorf("ComputeCumulativeDiff ReadBlock Not found")
		}
		block.SetMutableInfo(mutableInfo)

		if (block.GetStatus() & types.BlockTmpMaxDiff) != 0 {
			if !IsTheSameTimeSlice(toBeAddedBlock.GetTime(), block.GetTime()) {
				singleChainLinkInfo.set(true, hash, big0.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff()))
			} else {
				if block.GetDiff().Cmp(toBeAddedBlock.GetDiff()) < 0 {
					singleChainLinkInfo.set(true, block.GetMaxLink(), big0.Add(big0.Sub(block.GetCumulativeDiff(), block.GetDiff()), toBeAddedBlock.GetDiff()))
				} else {
					singleChainLinkInfo.set(false, hash, block.GetCumulativeDiff())
				}
			}
		} else {
			if !IsTheSameTimeSlice(toBeAddedBlock.GetTime(), block.GetTime()) {
				singleChainLinkInfo.set(true, block.GetMaxLink(), big0.Add(block.GetCumulativeDiff(), toBeAddedBlock.GetDiff()))
			} else {
				DiffBefore := utils.CalculateWork(block.GetMaxLink())
				if DiffBefore.Cmp(toBeAddedBlock.GetDiff()) < 0 {
					mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, block.GetMaxLink())
					if err != nil {
						return false, err
					}
					singleChainLinkInfo.set(true, mutableInfo.MaxLinkHash, big0.Add(big0.Sub(block.GetCumulativeDiff(), DiffBefore), toBeAddedBlock.GetDiff()))
				} else {
					singleChainLinkInfo.set(false, block.GetMaxLink(), block.GetCumulativeDiff())
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

//findTheMainBlockThatCanConfirmOtherBlocks find the main block that can confirm other blocks
func (mainChain *MainChain) findTheMainBlockThatCanConfirmOtherBlocks(mainTail *types.TailMainBlockInfo) ([]types.Block, types.Block, error) {
	tail := mainChain.GetTail()
	hash := tail.Hash
	now := utils.GetTimeStamp()
	var (
		canConfirm   bool
		listBlock    []types.Block
		locatorBlock types.Block
	)
	for {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return nil, nil, fmt.Errorf("findTheMainBlockThatCanConfirmOtherBlocks ReadBlockMutableInfo Not found. hash=%s", hash.String())
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return nil, nil, fmt.Errorf("findTheMainBlockThatCanConfirmOtherBlocks ReadBlock Not found. hash=%s", hash.String())
		}
		block.SetMutableInfo(mutableInfo)

		if !canConfirm && TimeSliceDifference(now, block.GetTime()) > params.ConfirmBlock {
			canConfirm = true
		}

		if canConfirm {
			if ((block.GetStatus() & types.BlockMain) != 0) && (block.GetTime() <= mainTail.Time) {
				locatorBlock = block
				break
			}
			listBlock = append([]types.Block{block}, listBlock...)
			log.Debug("findTheMainBlockThatCanConfirmOtherBlocks\n" + block.String())
		}
		hash = block.GetMutableInfo().MaxLinkHash
	}
	return listBlock, locatorBlock, nil
}

func (mainChain *MainChain) Confirm() error {
	mainTail := mainChain.GetMainTail()
	number := mainTail.Number
	listMainBlock, locatorBlock, err := mainChain.findTheMainBlockThatCanConfirmOtherBlocks(mainTail)
	if err != nil {
		return err
	}

	if locatorBlock.GetTime() < mainTail.Time {
		hash := mainTail.Hash
		for {
			mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
			if err != nil {
				return fmt.Errorf("confirm ReadBlockMutableInfo Not found. hash=%s", hash.String())
			}
			block := storage.ReadBlock(mainChain.db, hash)
			if block == nil {
				return fmt.Errorf("confirm ReadBlock Not found. hash=%s", hash.String())
			}
			block.SetMutableInfo(mutableInfo)

			if locatorBlock.GetTime() >= block.GetTime() {
				if locatorBlock.GetHash() != block.GetHash() {
					log.Error("locatorBlock!=block")
				}
				break
			}

			if err = mainChain.RollBackStatus(block.GetHash(), number); err != nil {
				return err
			}
			number--
			hash = block.GetMaxLink()
		}
	}
	if mainTail.Number > number {
		log.Warn("branching", "RollBack count", mainTail.Number-number)
	}

	if len(listMainBlock) == 0 {
		return nil
	}

	mainBlock, err := storage.ReadMainBlock(mainChain.db, number)
	if err != nil {
		return fmt.Errorf("%d ReadMainBlock lastMainTimeSlice fail", number)
	}
	state, err := state.New(mainBlock.Root, mainChain.stateDb)
	if err != nil {
		return fmt.Errorf("%s state.New fail [%s]", mainBlock.Root.String(), err.Error())
	}

	for index, block := range listMainBlock {
		log.Debug("the main block confirm\n"+block.String(), "index", index)
		number++
		info := block.GetMutableInfo()
		info.Status |= types.BlockMain
		block.SetMutableInfo(info)
		mainChain.CurrentMainBlock = block
		count := uint64(0)
		confirmReward, err := mainChain.singleBlockConfirm(block, block, number, &count, state)
		if err != nil {
			return err
		}
		if err := storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), info); err != nil {
			return err
		}

		log.Debug("the main block confirm finished", "confirmReward.userReward", confirmReward.userReward, "confirmReward.minerReward", confirmReward.minerReward)

		if err := CalculatingMinerReward(block, number, confirmReward, state); err != nil {
			return err
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
		err = storage.WriteMainBlock(mainChain.db, &types.MainBlockInfo{Hash: block.GetHash(), Root: root, ConfirmCount: count}, number)
		if err != nil {
			return err
		}

		mainChain.mainTailRWLock.Lock()
		log.Debug("begin update MainTail\n"+block.String(), "MainTail", mainChain.MainTail.String())
		mainChain.MainTail.Hash = block.GetHash()
		mainChain.MainTail.Time = block.GetTime()
		mainChain.MainTail.Number = number
		mainChain.MainTail.CumulativeDiff = block.GetCumulativeDiff()
		mainChain.lastState = state
		if err := storage.WriteTailMainBlockInfo(mainChain.db, &mainChain.MainTail); err != nil {
			log.Error("Confirm WriteTailMainBlockInfo", "error", err)
		}
		log.Debug("update MainTail finished", "MainTail", mainChain.MainTail.String())
		mainChain.mainTailRWLock.Unlock()
	}
	return nil
}

func (mainChain *MainChain) RollBackStatus(hash common.Hash, number uint64) error {
	log.Debug("RollBackStatus main block", "hash", hash)
	block := storage.ReadBlock(mainChain.db, hash)
	if block == nil {
		return fmt.Errorf("RollBackStatus ReadBlock Not found")
	}
	mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
	if err != nil {
		return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
	}
	log.Debug("RollBackStatus main block\n"+block.String(), "mutableInfo", mutableInfo)

	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsNumber < number {
			continue
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = mainChain.singleRollBackStatus(block, number); err != nil {
			return err
		}
	}

	mutableInfo.Status &= ^(types.BlockMain | types.BlockConfirm | types.BlockApply)
	mutableInfo.ConfirmItsNumber = 0
	mutableInfo.ConfirmItsIndex = 0
	if err = storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), mutableInfo); err != nil {
		return fmt.Errorf("RollBackStatus err:" + err.Error())
	}
	from, _ := block.GetSender()
	mainChain.LocalBlockNoticeSend(block.GetHash(), from, RollBackAction)
	return nil
}

func (mainChain *MainChain) singleRollBackStatus(block types.Block, number uint64) error {
	log.Debug("singleRollBackStatus\n" + block.String())
	for _, hash := range block.GetLinks() {
		mutableInfo, err := storage.ReadBlockMutableInfo(mainChain.db, hash)
		if err != nil {
			return fmt.Errorf("hash=%s ReadBlockMutableInfo fail. %s", hash.String(), err.Error())
		}
		if mutableInfo.ConfirmItsNumber < number {
			continue
		}
		block := storage.ReadBlock(mainChain.db, hash)
		if block == nil {
			return fmt.Errorf("hash=%s ReadBlock Not found", hash.String())
		}
		block.SetMutableInfo(mutableInfo)
		if err = mainChain.singleRollBackStatus(block, number); err != nil {
			return err
		}
	}
	mutableInfo := block.GetMutableInfo()
	mutableInfo.Status &= ^(types.BlockConfirm | types.BlockApply)
	mutableInfo.ConfirmItsNumber = 0
	mutableInfo.ConfirmItsIndex = 0
	storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), mutableInfo)
	from, _ := block.GetSender()
	mainChain.LocalBlockNoticeSend(block.GetHash(), from, RollBackAction)
	return nil
}

func (mainChain *MainChain) singleBlockConfirm(block types.Block, mainBlock types.Block, number uint64, count *uint64, state *state.StateDB) (*ConfirmRewardInfo, error) {
	var (
		confirmRewardInfo ConfirmRewardInfo
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
		linksReward, err := mainChain.singleBlockConfirm(block, mainBlock, number, count, state)
		if err != nil {
			log.Error(err.Error())
		}
		log.Debug("singleBlockConfirm ancestors finished", "hash", block.GetHash().String(), "linksReward.userReward", linksReward.userReward, "linksReward.minerReward", linksReward.minerReward)
		if linksReward != nil {
			confirmRewardInfo.userReward.Add(confirmRewardInfo.userReward, linksReward.userReward)
			confirmRewardInfo.minerReward.Add(confirmRewardInfo.minerReward, linksReward.minerReward)
		}
	}
	log.Debug("singleBlockConfirm self\n"+block.String(), "confirmRewardInfo.userReward", confirmRewardInfo.userReward.String(), "confirmRewardInfo.minerReward", confirmRewardInfo.minerReward.String())
	*count++
	block.GetMutableInfo().ConfirmItsNumber = number
	block.GetMutableInfo().Status |= types.BlockConfirm
	block.GetMutableInfo().ConfirmItsIndex = *count

	var err error
	var receipt *types.Receipt
	confirmRewardInfo.userReward, receipt, err = mainChain.CalculatingAccounts(block, confirmRewardInfo.userReward, count, state)
	if err == nil {
		block.GetMutableInfo().Status |= types.BlockApply
	}

	if err = storage.WriteBlockMutableInfo(mainChain.db, block.GetHash(), block.GetMutableInfo()); err != nil {
		log.Error(err.Error())
	}

	from, _ := block.GetSender()
	mainChain.LocalBlockNoticeSend(block.GetHash(), from, ConfirmAction)
	mainChain.sendStatusInfoToCenter(block, mainBlock, receipt)

	log.Debug(block.String())
	return ComputeConfirmReward(&confirmRewardInfo), err
}

func (mainChain *MainChain) sendStatusInfoToCenter(block types.Block, mainBlock types.Block, receipt *types.Receipt) {
	if mainChain.mq == nil {
		return
	}
	IsMain := "0"
	if block == mainBlock {
		IsMain = "1"
	}

	var (
		GasUsed, ConfirmedOrder string
	)

	if receipt != nil {
		GasUsed = strconv.FormatUint(receipt.CumulativeGasUsed, 10)
		ConfirmedOrder = strconv.FormatUint(receipt.Index, 10)
	}

	ConfirmStatus := "apply"
	if block.GetStatus()&types.BlockApply == 0 {
		ConfirmStatus = "reject"
	}

	blockStatus := types.MQBlockStatus{
		BlockHash:      block.GetHash().String(),
		BlockHigh:      strconv.FormatUint(block.GetMutableInfo().ConfirmItsNumber, 10),
		IsMain:         IsMain,
		ConfirmStatus:  ConfirmStatus,
		ConfirmDate:    strconv.FormatUint(utils.GetTimeStamp(), 10),
		GasUsed:        GasUsed,
		ConfirmedHash:  mainBlock.GetHash().String(),
		ConfirmedHigh:  strconv.FormatUint(block.GetMutableInfo().ConfirmItsNumber, 10),
		ConfirmedOrder: ConfirmedOrder,
	}
	if err := mainChain.mq.Publish("blockStatus", blockStatus); err != nil {
		log.Error(err.Error())
	}
}
