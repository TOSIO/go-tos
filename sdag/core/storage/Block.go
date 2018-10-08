package storage

import (
	"fmt"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/devbase/utils"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

func ReadBlockRlp(db Reader, hash common.Hash) []byte {
	data, _ := db.Get(hash.Bytes())
	return data
}

func ReadBlock(db Reader, hash common.Hash) types.Block {
	data := ReadBlockRlp(db, hash)
	if len(data) == 0 {
		return nil
	}

	block, err := types.BlockUnRlp(data)
	if err != nil {
		log.Error("Invalid block RLP", "hash", hash, "err", err)
	}

	return block
}

func HasBlock(db Reader, hash common.Hash) bool {
	if has, err := db.Has(hash.Bytes()); !has || err != nil {
		return false
	}
	return true
}

func WriteBlockRlp(db Writer, hash common.Hash, time uint64, blockRLP []byte) error {

	if err := db.Put(blockKey(utils.GetMainTime(time)), hash.Bytes()); err != nil {
		log.Error("Failed to store block index", "err", err)
		return err
	}

	if err := db.Put(hash.Bytes(), blockRLP); err != nil {
		log.Error("Failed to store block", "err", err)
		return err
	}

	return nil
}

func WriteBlock(db Writer, block types.Block) error {
	return WriteBlockRlp(db, block.GetHash(), block.GetTime(), block.GetRlp())
}

// 根据指定的时间片获取对应的所有区块hash
func ReaderBlockHashByTmSlice(db ReadIteration, slice uint64) ([]common.Hash, error) {
	//mTime := utils.GetMainTime(slice)
	var hashes []common.Hash
	key := blockLookUpKey(slice)
	iterator := db.NewIteratorWithPrefix(key)
	for iterator.Next() {
		fmt.Printf("key:%s, value:%s\n", iterator.Key(), iterator.Value())
		hash := common.BytesToHash(iterator.Value())
		hashes = append(hashes, hash)
	}

	return hashes, nil
}

//根据指定的hash集合返回对应的区块（RLP流）
func ReadBlocks(db Reader, hashes []common.Hash) ([][]byte, error) {

	var blockRlps [][]byte
	for _, hash := range hashes {
		blockRlp := ReadBlockRlp(db, hash)
		if blockRlp != nil {
			blockRlps = append(blockRlps, blockRlp)
		}
	}

	return blockRlps, nil
}

func Update(db ReaderWrite, hash common.Hash, data interface{}, update func(block types.Block, data interface{})) error {
	linkBlockEI := ReadBlock(db, hash) //the 'EI' is empty interface logogram
	if linkBlockEI == nil {
		if block, ok := linkBlockEI.(types.Block); ok {
			update(block, data)
		} else {
			log.Error("linkBlockEI assertion failure")
			return fmt.Errorf("linkBlockEI assertion failure")
		}
	} else {
		log.Error("GetBlock fail")
		return fmt.Errorf("GetBlock fail")
	}
	return nil
}

func ReadBlockMutableInfoRlp(db Reader, hash common.Hash) ([]byte, error) {
	data, err := db.Get(blockInfoKey(hash))
	return data, err
}

func WriteBlockMutableInfoRlp(db Writer, hash common.Hash, blockMutableInfoRLP []byte) error {
	if err := db.Put(blockInfoKey(hash), blockMutableInfoRLP); err != nil {
		log.Error("Failed to store block info", "err", err)
		return err
	}

	return nil
}
