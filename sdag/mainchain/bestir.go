package mainchain

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/params"
	"github.com/TOSIO/go-tos/sdag/core"
	"github.com/TOSIO/go-tos/sdag/core/state"
	"github.com/TOSIO/go-tos/sdag/core/storage"
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

func (mainChain *MainChain) CalculatingAccounts(block types.Block, confirmReward *big.Int, count *uint64, state *state.StateDB) (*big.Int, *types.Receipt, error) {
	log.Debug("begin CalculatingAccounts", "hash", block.GetHash().String())
	address, err := block.GetSender()
	if err != nil {
		log.Error(err.Error())
		return confirmReward, nil, err
	}
	log.Debug("Add confirmReward", "address", address, "confirmReward", confirmReward.String())

	state.AddBalance(address, confirmReward)
	balance := state.GetBalance(address)
	log.Debug("Calculate the balance after confirming the reward", "address", address, "balance", balance.String())
	var gasPool uint64
	if new(big.Int).Mul(big.NewInt(int64(block.GetGasLimit())), block.GetGasPrice()).Cmp(balance) > 0 {
		gasPool = new(big.Int).Quo(balance, block.GetGasPrice()).Uint64()
	} else {
		gasPool = block.GetGasLimit()
	}

	cost, receipt, err := mainChain.ExecutionTransaction(block, gasPool, count, state)
	log.Debug("ComputeCost", "block", block.GetHash().String(), "cost", cost.String())

	actualCostDeduction := deductCost(address, balance, cost, state)
	log.Debug("deduct cost", "address", address, "actualCostDeduction", actualCostDeduction.String(), "balance", state.GetBalance(address))

	return actualCostDeduction, receipt, err
}

func (mainChain *MainChain) ExecutionTransaction(block types.Block, gasPool uint64, count *uint64, state *state.StateDB) (*big.Int, *types.Receipt, error) {
	needUseGas, IsGasCostOverrun, receipt, err := mainChain.TransferAccounts(block, gasPool, state)
	actualGas := needUseGas
	var returnErr error
	var receiptStatus uint64
	if err != nil {
		err = fmt.Errorf("ExecutionTransaction error:" + err.Error())
		returnErr = err
	} else if IsGasCostOverrun {
		returnErr = fmt.Errorf("gas use overrun")
		actualGas = gasPool
	}

	if (block.GetMutableInfo().Status & types.BlockMain) != 0 {
		actualGas = 0
		needUseGas = 0
	}

	if returnErr == nil {
		receiptStatus = types.ReceiptStatusSuccessful
	} else {
		receiptStatus = types.ReceiptStatusFailed
	}

	if receipt == nil {
		receipt = &types.Receipt{
			TxHash:  block.GetHash(),
			Status:  receiptStatus,
			GasUsed: actualGas,
			Index:   *count,
		}
	}
	receipt.Index = *count
	err = storage.WriteReceiptInfo(mainChain.db, block.GetHash(), receipt)
	if err != nil {
		log.Error("ExecutionTransaction error:" + err.Error())
	}

	return ComputeTransactionCost(needUseGas, block.GetGasPrice()), receipt, returnErr
}

func (mainChain *MainChain) TransferAccounts(block types.Block, gasPool uint64, state *state.StateDB) (uint64, bool, *types.Receipt, error) {
	var (
		costGas uint64
		err     error
		receipt *types.Receipt
	)

	mainAddress, err := mainChain.CurrentMainBlock.GetSender()
	if err != nil {
		log.Error("ComputeCostGas err:" + err.Error())
		return costGas, gasPool < costGas, nil, fmt.Errorf("ComputeCostGas err:" + err.Error())
	}

	if block.GetType() == types.BlockTypeTx {
		costGas = params.TxGas
		txBlock, ok := block.(*types.TxBlock)
		if ok {
			costGas *= uint64(len(txBlock.Outs))
			GasPool := core.GasPool(gasPool)
			if len(txBlock.GetPayload()) != 0 {
				var contractGas uint64
				receipt, costGas, err = core.ApplyTransaction(mainChain.ChainConfig, mainChain, &mainAddress,
					&GasPool, state, txBlock, &contractGas, mainChain.VMConfig)
			} else {
				from, _ := block.GetSender()
				balance := state.GetBalance(from)
				state.SetNonce(from, state.GetNonce(from)+1)
				if (block.GetMutableInfo().Status & types.BlockMain) != 0 {
					costGas = 0
				}
				if !CheckTransactionAmount(balance, txBlock.Outs, ComputeTransactionCost(costGas, block.GetGasPrice())) {
					err = fmt.Errorf("%s insufficient balance", from.String())
					log.Info(err.Error())
					return costGas, gasPool < costGas, nil, err
				}

				for _, out := range txBlock.Outs {
					log.Debug("calculate transaction", "address", from, "out", out)
					log.Debug("Receiver balance", "Receiver", out.Receiver, "balance", state.GetBalance(out.Receiver))
					state.SubBalance(from, out.Amount)
					state.AddBalance(out.Receiver, out.Amount)
					log.Debug("after calculate transaction", "address", from, "balance", balance.String(), "Receiver", out.Receiver, "Receiver balance", state.GetBalance(out.Receiver))
				}
			}
		} else {
			err = fmt.Errorf("block.(*types.TxBlock) assert fail")
			log.Error("TransferAccounts:" + err.Error())
			return costGas, gasPool < costGas, nil, err
		}
	} else {
		costGas = params.MiningGasUsed
	}

	return costGas, gasPool < costGas, receipt, err
}

func ComputeTransactionCost(costGas uint64, gasPrice *big.Int) *big.Int {
	return new(big.Int).Mul(gasPrice, big.NewInt(int64(costGas)))
}

func deductCost(address common.Address, balance *big.Int, cost *big.Int, state *state.StateDB) *big.Int {
	balance = new(big.Int).Set(balance)
	if balance.Cmp(cost) < 0 {
		state.SetBalance(address, big.NewInt(0))
		return balance
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
