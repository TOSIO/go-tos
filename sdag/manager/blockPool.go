package manager

import (
	"container/list"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

var (
	UnverifiedTransactionList *list.List
	MaxConfirm                = 11
)

func init() {
	UnverifiedTransactionList = list.New()
}

func AddBlock(emptyInterfaceBlock interface{}) error {
	var (
		block types.Block
		ok    bool
	)
	if block, ok = emptyInterfaceBlock.(types.Block); ok {
		block.Validation()

		_, err := storage.GetBlock(block.GetHash())
		if err != nil {
			log.Error("The block has been added")
			return fmt.Errorf("The block has been added")
		} else {
			log.Info("Non-repeating block")
		}
	} else {
		log.Error("addBlock block.(types.Block) error")
		return fmt.Errorf("addBlock block.(types.Block) error")
	}

	var linkHaveIsolated bool

	for _, hash := range block.GetLinks() {
		linkBlockHash, err := storage.GetBlock(hash)
		if err == nil {
			if linkBlockI, ok := linkBlockHash.(types.Block); ok {
				if linkBlockI.GetTime() > block.GetTime() {
					log.Error("links time error")
					return fmt.Errorf("links time error")
				} else {
					log.Info("links time legal")
				}
			}
		} else {
			linkHaveIsolated = true
			//TODO
			//add Isolated block
		}
	}

	if linkHaveIsolated {
		//TODO
		//add Isolated block
	} else {
		for _, hash := range block.GetLinks() {
			linkBlockHash, err := storage.GetBlock(hash)
			if err == nil {
				if linkBlockI, ok := linkBlockHash.(types.Block); ok {
					linkBlockI.SetStatus(linkBlockI.GetStatus() | types.BlockVerify)
				}
			} else {
				linkHaveIsolated = true
				//TODO
				//add Isolated block
			}
		}
	}

	log.Warn("%s is a  Isolated block", block.GetHash())
	log.Info("Verify before adding a block")

	storage.PutBlock(block.GetHash(), block.GetRlp())
	addUnverifiedTransactionList(block.GetHash())
	return nil
}

func PopUnverifiedTransactionList() (interface{}, error) {
	if UnverifiedTransactionList.Len() > 0 {
		return UnverifiedTransactionList.Remove(UnverifiedTransactionList.Front()), nil
	} else {
		return nil, fmt.Errorf("the list is empty")
	}
}

func Confirm(links []common.Hash) {
	listLen := UnverifiedTransactionList.Len()
	for i := 0; i < listLen && i < MaxConfirm; i++ {
		hashI, err := PopUnverifiedTransactionList()
		if err != nil {
			log.Error(err.Error())
		}
		hash, ok := hashI.(common.Hash)
		if !ok {
			log.Error("hash.(common.Hash)")
		}
		links = append(links, hash)
	}
}

func addUnverifiedTransactionList(v interface{}) {
	UnverifiedTransactionList.PushFront(v)
}
