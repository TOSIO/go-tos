// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"github.com/TOSIO/go-tos/devbase/common"

	"github.com/TOSIO/go-tos/sdag/core/types"
	"github.com/TOSIO/go-tos/sdag/core/vm"
)

// ChainContext supports retrieving headers and consensus parameters from the
// current blockchain to be used during transaction processing.
type ChainContext interface {
	// Engine retrieves the chain's consensus engine.
	//Engine() consensus.Engine

	// GetHeader returns the hash corresponding to their hash.
	//GetHeader(common.Hash, uint64) *types.Header
	GetMainBlock(uint64) types.Block
}

// NewEVMContext creates a new context for use in the EVM.
func NewEVMContext(msg Message, block types.Block, chain ChainContext, author *common.Address) vm.Context {
	// If we don't have an explicit author (i.e. not mining), extract from the header
	//var beneficiary common.Address
	/* 	if author == nil {
	   		beneficiary, _ = chain.Engine().Author(header) // Ignore error, we're past header validation
	   	} else {
	   		beneficiary = *author
		   } */
	number := &big.Int{}
	number.SetUint64(block.GetMutableInfo().ConfirmItsNumber)
	return vm.Context{
		CanTransfer: CanTransfer,
		Transfer:    Transfer,
		GetHash:     GetHashFn( /* tx,  */ chain),
		Origin:      msg.From(),
		Coinbase:    *author,
		BlockNumber: new(big.Int).Set(number),
		Time:        new(big.Int).SetUint64(block.GetTime()),
		Difficulty:  new(big.Int).Set(block.GetDiff()),
		GasLimit:    block.GetGasLimit(),
		GasPrice:    new(big.Int).Set(msg.GasPrice()),
	}
}

// GetHashFn returns a GetHashFunc which retrieves header hashes by number
func GetHashFn( /* ref *types.TxBlock,  */ chain ChainContext) func(n uint64) common.Hash {
	//var cache map[uint64]common.Hash

	return func(n uint64) common.Hash {
		if chain != nil {
			if block := chain.GetMainBlock(n); block != nil {
				return block.GetHash()
			} else {
				return common.Hash{}
			}
		}
		return common.Hash{}

		//If there's no hash cache yet, make one
		//if cache == nil {
		//	cache = map[uint64]common.Hash{
		//		ref.Number - 1: hash,
		//	}
		//}
		//// Try to fulfill the request from the cache
		//if hash, ok := cache[n]; ok {
		//	return hash
		//}
		//// Not cached, iterate the blocks and cache the hashes
		//for header := chain.GetMainBlock(ref.Number - 1); header != nil; header = chain.GetMainBlock(header.Number - 1) {
		//	cache[header.Number-1] = header.GetHash()
		//	if n == header.Number-1 {
		//		return header.GetHash()
		//	}
		//}
		//return common.Hash{}
	}
}

// CanTransfer checks whether there are enough funds in the address' account to make a transfer.
// This does not take the necessary gas in to account to make the transfer valid.
func CanTransfer(db vm.StateDB, addr common.Address, amount *big.Int) bool {
	return db.GetBalance(addr).Cmp(amount) >= 0
}

// Transfer subtracts amount from sender and adds amount to recipient using the given Db
func Transfer(db vm.StateDB, sender, recipient common.Address, amount *big.Int) {
	db.SubBalance(sender, amount)
	db.AddBalance(recipient, amount)
}
