package manager

import (
	"container/list"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/storage/tosdb"
	"github.com/TOSIO/go-tos/sdag/core"
	"time"

	"github.com/TOSIO/go-tos/devbase/common"
	"github.com/TOSIO/go-tos/devbase/log"
	"github.com/TOSIO/go-tos/sdag/core/storage"
	"github.com/TOSIO/go-tos/sdag/core/types"
)

var (
	UnverifiedTransactionList *list.List
	MaxConfirm                = 4
)

type protocolManagerI interface {
	RelayBlock(blockRLP []byte) error
}

var (
	pm protocolManagerI
	db tosdb.Database
)

func init() {
	UnverifiedTransactionList = list.New()
	UnverifiedTransactionList.PushFront(GetMainBlockTail())
}

func SetDB(chainDb tosdb.Database) {
	db = chainDb
}

func GetMainBlockTail() common.Hash {
	var hash common.Hash
	hash = core.GenesisHash
	return hash
}

func SetProtocolManager(protocolManager protocolManagerI) {
	pm = protocolManager
}

type IsolatedBlock struct {
	Links  []common.Hash //Links
	LinkIt []common.Hash //Link it
	Time   uint32
	RLP    []byte
}

type lackBlock struct {
	LinkIt []common.Hash //Link it
	Time   uint32
}

var IsolatedBlockMap = make(map[common.Hash]IsolatedBlock)
var lackBlockMap = make(map[common.Hash]lackBlock)

func addIsolatedBlock(block types.Block, links []common.Hash) {
	if _, ok := IsolatedBlockMap[block.GetHash()]; ok {
		log.Warn("the Isolated block already exists")
		return
	}

	IsolatedBlockMap[block.GetHash()] = IsolatedBlock{links, []common.Hash{}, uint32(time.Now().Unix()), block.GetRlp()}
	for _, link := range links {
		v, ok := IsolatedBlockMap[link]
		if ok {
			v.LinkIt = append(v.LinkIt, link)
			IsolatedBlockMap[link] = v
		} else {
			v, ok := lackBlockMap[link]
			if ok {
				v.LinkIt = append(v.LinkIt, link)
				lackBlockMap[link] = v
			} else {
				lackBlockMap[link] = lackBlock{[]common.Hash{link}, uint32(time.Now().Unix())}
			}
		}
	}

	lackBlock, ok := lackBlockMap[block.GetHash()]
	if ok {
		v := IsolatedBlockMap[block.GetHash()]
		v.LinkIt = append(v.LinkIt, lackBlock.LinkIt...)
		IsolatedBlockMap[block.GetHash()] = v
		delete(lackBlockMap, block.GetHash())
	}
}

func deleteIsolatedBlock(block types.Block) {
	v, ok := lackBlockMap[block.GetHash()]
	if ok {
		delete(lackBlockMap, block.GetHash())
		currentList := v.LinkIt
		nextList := []common.Hash{}
		for len(currentList) > 0 {
			for _, hash := range currentList {
				linkBlock := IsolatedBlockMap[hash]
				if len(linkBlock.Links) == 1 {
					newBlock, err := types.BlockUnRlp(linkBlock.RLP)
					if err != nil {
						linkCheckAndSave(newBlock)
					} else {
						log.Error("BlockUnRlp fail")
					}
					nextList = append(nextList, linkBlock.LinkIt...)
					delete(IsolatedBlockMap, hash)
				} else {
					linkBlock.Links = deleteLinkHash(linkBlock.Links, hash)
				}
			}
			currentList = nextList
			nextList = []common.Hash{}
		}
	}
}

func deleteLinkHash(link []common.Hash, hash common.Hash) []common.Hash {
	for i, v := range link {
		if v == hash {
			link = append(link[:i], link[i+1:]...)
			return link
		}
	}
	log.Error("Hash not found")
	return link
}

func AddBlock(block types.Block) error {
	err := block.Validation()
	if err != nil {
		log.Error("the block Validation fail")
		return fmt.Errorf("the block Validation fail")
	}

	ok := storage.HasBlock(db, block.GetHash())
	if ok {
		log.Error("the block has been added")
		return fmt.Errorf("the block has been added")
	} else {
		log.Info("Non-repeating block")
	}

	err = linkCheckAndSave(block)
	deleteIsolatedBlock(block)
	pm.RelayBlock(block.GetRlp())
	return err
}

func linkCheckAndSave(block types.Block) error {
	var linkHaveIsolated bool
	var linkBlockIs []types.Block
	var linksLackBlock []common.Hash

	for _, hash := range block.GetLinks() {
		if hash == core.GenesisHash {

		} else {
			linkBlockEI := storage.ReadBlock(db, hash) //the 'EI' is empty interface logogram
			if linkBlockEI != nil {
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
				linksLackBlock = append(linksLackBlock, hash)
			}
		}
	}

	if linkHaveIsolated {
		log.Warn(block.GetHash().String() + "is a Isolated block")
		addIsolatedBlock(block, linksLackBlock)
	} else {
		log.Info("Verification passed")
		for _, linkBlockI := range linkBlockIs {
			linkBlockI.SetStatus(linkBlockI.GetStatus() | types.BlockVerify)
			storage.WriteBlockMutableInfoRlp(db, linkBlockI.GetHash(), types.GetMutableRlp(linkBlockI.GetMutableInfo()))
		}

		storage.WriteBlock(db, block)
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

func Confirm(links []common.Hash) []common.Hash {
	listLen := UnverifiedTransactionList.Len()
	for i := 0; i < listLen && i < MaxConfirm; i++ {
		hashI, err := PopUnverifiedTransactionList()
		if err != nil {
			log.Error(err.Error())
		}
		hash, ok := hashI.(common.Hash)
		if !ok {
			log.Error("error hash.(common.Hash)")
		}
		links = append(links, hash)
	}
	return links
}

func addUnverifiedTransactionList(v interface{}) {
	UnverifiedTransactionList.PushFront(v)
}
