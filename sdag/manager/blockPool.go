package manager

import (
	"container/list"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
	"time"
)

var (
	UnverifiedTransactionList *list.List
	MaxConfirm                = 11
)

func init() {
	UnverifiedTransactionList = list.New()
}

type IsolatedBlock struct {
	Links  []common.Hash //Links
	linkIt []common.Hash //Link it
	time   uint32
}

type lackBlock struct {
	linkIt []common.Hash //Link it
	time   uint32
}

var IsolatedBlockMap = make(map[common.Hash]IsolatedBlock)
var lackBlockMap = make(map[common.Hash]lackBlock)

func addIsolatedBlock(block types.Block) {
	IsolatedBlockMap[block.GetHash()] = IsolatedBlock{block.GetLinks(), []common.Hash{}, uint32(time.Now().Unix())}
	for _, link := range block.GetLinks() {
		v, ok := IsolatedBlockMap[link]
		if ok {
			v.linkIt = append(v.linkIt, link)
		} else {
			v, ok := lackBlockMap[link]
			if ok {
				v.linkIt = append(v.linkIt, link)
			} else {
				lackBlockMap[link] = lackBlock{[]common.Hash{link}, uint32(time.Now().Unix())}
			}
		}
	}
}

func AddBlock(emptyInterfaceBlock interface{}) error {
	var (
		block types.Block
		ok    bool
	)
	if block, ok = emptyInterfaceBlock.(types.Block); ok {
		err := block.Validation()
		if err != nil {
			log.Error("the block Validation fail")
			return fmt.Errorf("the block Validation fail")
		}
		_, err = storage.GetBlock(block.GetHash())
		if err != nil {
			log.Error("the block has been added")
			return fmt.Errorf("the block has been added")
		} else {
			log.Info("Non-repeating block")
		}
	} else {
		log.Error("addBlock block.(types.Block) error")
		return fmt.Errorf("addBlock block.(types.Block) error")
	}

	var linkHaveIsolated bool
	var linkBlockIs []types.Block

	for _, hash := range block.GetLinks() {
		linkBlockEI, err := storage.GetBlock(hash) //the 'EI' is empty interface logogram
		if err == nil {
			if linkBlockI, ok := linkBlockEI.(types.Block); ok {
				if linkBlockI.GetTime() > block.GetTime() {
					log.Error("links time error")
					return fmt.Errorf("links time error")
				} else {
					linkBlockIs = append(linkBlockIs, linkBlockI)
					log.Info("links time legal")
				}
			} else {
				log.Error("linkBlockEI assertion failure")
				return fmt.Errorf("linkBlockEI assertion failure")
			}
		} else {
			linkHaveIsolated = true
			//TODO
			//add Isolated block
		}
	}

	if linkHaveIsolated {
		log.Warn("%s is a  Isolated block", block.GetHash())
		block.SetStatus(block.GetStatus() | types.BlockVerify)
		//TODO
		//add Isolated block
	} else {
		for _, linkBlockI := range linkBlockIs {
			linkBlockI.SetStatus(linkBlockI.GetStatus() | types.BlockVerify)
			storage.PutBlock(linkBlockI.GetHash(), linkBlockI.GetAllRlp())
		}
	}

	log.Info("Verification passed")

	storage.PutBlock(block.GetHash(), block.GetRlp())
	if !linkHaveIsolated {
		addUnverifiedTransactionList(block.GetHash())
	}
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
