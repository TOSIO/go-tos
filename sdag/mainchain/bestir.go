package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"math"
	"math/big"
)

type ConfirmRewardInfo struct {
	userReward  *big.Int
	minerReward *big.Int
}

func (reward *ConfirmRewardInfo) Init() {
	reward.userReward = big.NewInt(0)
	reward.minerReward = big.NewInt(0)
}

func CalculatingAccounts(block types.Block, confirmReward *big.Int, state *state.StateDB) (*big.Int, error) {
	log.Debug("begin CalculatingAccounts", "hash", block.GetHash().String())
	cost, IsGasCostOverrun, _ := ComputeCost(block)
	log.Debug("ComputeCost", "block", block.GetHash().String(), "cost", cost.String(), "IsGasCostOverrun", IsGasCostOverrun)
	address, err := block.GetSender()
	if err != nil {
		log.Error(err.Error())
		return big.NewInt(0), err
	}
	log.Debug("Add confirmReward", "address", address, "confirmReward", confirmReward.String())

	state.AddBalance(address, confirmReward)
	balance := state.GetBalance(address)
	log.Debug("Calculate the balance after confirming the reward", "address", address, "balance", balance.String())

	actualCostDeduction := deductCost(address, balance, cost, state)
	log.Debug("deduct cost", "address", address, "actualCostDeduction", actualCostDeduction.String(), "balance", state.GetBalance(address))
	if block.GetType() == types.BlockTypeTx {
		if !IsGasCostOverrun {
			txBlock, ok := block.(*types.TxBlock)
			if !ok {
				err = fmt.Errorf("assert fail")
				log.Error(err.Error())
				return actualCostDeduction, err
			}

			if !CheckTransactionAmount(balance, txBlock.Outs, cost) {
				err = fmt.Errorf("%s insufficient balance", address.String())
				log.Info(err.Error())
				return actualCostDeduction, err
			}

			for _, out := range txBlock.Outs {
				log.Debug("calculate transaction", "address", address, "out", out)
				log.Debug("Receiver balance", "Receiver", out.Receiver, "balance", state.GetBalance(out.Receiver))
				state.SubBalance(address, out.Amount)
				state.AddBalance(out.Receiver, out.Amount)
				log.Debug("after calculate transaction", "address", address, "balance", balance.String(), "Receiver", out.Receiver, "Receiver balance", state.GetBalance(out.Receiver))
			}
		}
	}

	return actualCostDeduction, nil
}

func ComputeCost(block types.Block) (*big.Int, bool, error) {
	costGas, IsGasCostOverrun, err := ComputeCostGas(block)

	if IsGasCostOverrun {
		log.Info("gas cost overrun")
	}

	return ComputeTransactionCost(costGas, block.GetGasPrice()), IsGasCostOverrun, err
}

func ComputeCostGas(block types.Block) (uint64, bool, error) {
	var costGas uint64
	var IsGasCostOverrun bool
	var err error
	if (block.GetMutableInfo().Status & types.BlockMain) != 0 {
		return costGas, IsGasCostOverrun, err
	}
	if block.GetType() == types.BlockTypeTx {
		costGas = params.TransferTransactionGasUsed
		txBlock, ok := block.(*types.TxBlock)
		if ok {
			if len(txBlock.GetPayload()) != 0 {

			}
		} else {
			err = fmt.Errorf("block.(*types.TxBlock) error")
			log.Error(err.Error())
		}
	} else {
		costGas = params.MiningGasUsed
	}
	if costGas > block.GetGasLimit() {
		costGas = block.GetGasLimit()
		IsGasCostOverrun = true
	}
	return costGas, IsGasCostOverrun, err
}

func ComputeTransactionCost(costGas uint64, gasPrice *big.Int) *big.Int {
	return new(big.Int).Mul(gasPrice, big.NewInt(int64(costGas)))
}

func deductCost(address common.Address, balance *big.Int, cost *big.Int, state *state.StateDB) *big.Int {
	if balance.Cmp(cost) < 0 {
		state.SetBalance(address, big.NewInt(0))
		return new(big.Int).Set(balance)
	} else {
		state.SubBalance(address, cost)
		return new(big.Int).Set(cost)
	}
}

func CheckTransactionAmount(balance *big.Int, Outs []types.TxOut, cost *big.Int) bool {
	balance = new(big.Int).Set(balance)
	for _, out := range Outs {
		balance.Sub(balance, out.Amount)
		if balance.Sign() < 0 {
			return false
		}
	}
	balance.Sub(balance, cost)
	if balance.Sign() < 0 {
		return false
	}
	return true
}

func ComputeConfirmReward(reward *ConfirmRewardInfo) *ConfirmRewardInfo {
	temp := big.NewInt(0)
	temp.Div(temp.Mul(reward.userReward, big.NewInt(params.ConfirmUserRewardRate)), big.NewInt(100))
	reward.minerReward.Add(reward.minerReward, reward.userReward.Sub(reward.userReward, temp))
	return &ConfirmRewardInfo{temp, reward.minerReward}
}

func ComputeMainConfirmReward(reward *ConfirmRewardInfo) *big.Int {
	return new(big.Int).Add(reward.userReward, reward.minerReward)
}

func CalculatingMinerReward(block types.Block, blockNumber uint64, confirmRewardInfo *ConfirmRewardInfo, state *state.StateDB) error {
	address, err := block.GetSender()
	if err != nil {
		return err
	}
	RewardMiner := RewardMiner(blockNumber)
	state.AddBalance(address, new(big.Int).Add(ComputeMainConfirmReward(confirmRewardInfo), RewardMiner))
	log.Debug("CalculatingMinerReward", "RewardMiner", RewardMiner.String(),
		"confirmRewardInfo.userReward", confirmRewardInfo.userReward.String(), "confirmRewardInfo.minerReward", confirmRewardInfo.minerReward.String(),
		"address", address, "hash", block.GetHash().String())

	return nil
}

// RewardMiner implement 1/2^(n/HalfLife)*Initial
func RewardMiner(number uint64) *big.Int {
	return new(big.Int).Mul(big.NewInt(int64(math.Exp2(-float64(number/params.HalfLifeRewardMiner))*(params.InitialRewardMiner/params.OneTos))), big.NewInt(params.OneTos))
}
