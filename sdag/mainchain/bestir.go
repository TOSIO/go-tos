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
	reward.userReward = big.NewInt(0)
}

func CalculatingAccounts(block types.Block, confirmReward *big.Int, state *state.StateDB) (*big.Int, error) {
	cost, IsGasCostOverrun, _ := ComputeCost(block)
	address, err := block.GetSender()
	if err != nil {
		log.Error(err.Error())
		return big.NewInt(0), err
	}
	state.AddBalance(address, confirmReward)
	balance := state.GetBalance(address)
	actualDeduction := deductCost(address, balance, cost, state)
	if block.GetType() == types.BlockTypeTx {
		if !IsGasCostOverrun {
			txBlock, ok := block.(*types.TxBlock)
			if !ok {
				err = fmt.Errorf("assert fail")
				log.Error(err.Error())
				return actualDeduction, err
			}

			if CheckTransactionAmount(balance, txBlock.Outs, cost) {
				err = fmt.Errorf("insufficient balance")
				log.Error(err.Error())
				return actualDeduction, err
			}

			for _, out := range txBlock.Outs {
				state.SubBalance(address, out.Amount)
				state.AddBalance(out.Receiver, out.Amount)
			}
		}
	}

	return actualDeduction, nil
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
	return gasPrice.Mul(gasPrice, big.NewInt(int64(costGas)))
}

func deductCost(address common.Address, balance *big.Int, cost *big.Int, state *state.StateDB) *big.Int {
	if balance.Cmp(cost) < 0 {
		state.SetBalance(address, big.NewInt(0))
		return balance
	} else {
		state.SubBalance(address, cost)
		return cost
	}
}

func CheckTransactionAmount(balance *big.Int, Outs []types.TxOut, cost *big.Int) bool {
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
	return reward.userReward.Add(reward.userReward, reward.minerReward)
}

func CalculatingMinerReward(block types.Block, blockNumber uint64, state *state.StateDB) error {
	address, err := block.GetSender()
	if err != nil {
		return err
	}
	RewardMiner := RewardMiner(blockNumber)
	state.AddBalance(address, RewardMiner)

	return nil
}

//1/2^(n/HalfLife)*Initial
func RewardMiner(number uint64) *big.Int {
	return new(big.Int).Mul(big.NewInt(int64(math.Exp2(float64(-number/params.HalfLifeRewardMiner))*(params.InitialRewardMiner/params.OneTos))), big.NewInt(params.OneTos))
}
