package prque

//the reason that why i don't let push operator into  simple function then blow function
// can call is forward function call back can let gloabe p push many time same value

import (
	"fmt"
	"testing"
)

var sstackItems = []item{
	{"我是老三", 1},
	{"我是老二", 2},
	{"我是老大", 3},
}

func TestSstackPush(t *testing.T) {

	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	for _, itm := range sstackItems {
		ssk.Push(&itm)
	}

	if ssk.Len() != 3 {
		t.Error("push error!")
	}
}

func TestSstackPop(t *testing.T) {
	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	for _, itm := range sstackItems {
		ssk.Push(&itm)
	}
	getItem := ssk.Pop()

	if getItem.(*item).priority != 3 {
		t.Error("pop error")
	}
}

func TestSstackLen(t *testing.T) {

	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	for _, itm := range sstackItems {
		ssk.Push(&itm)
	}

	if ssk.Len() != 3 {
		t.Error("error")
	}
}

func TestSstackLess(t *testing.T) {

	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	blocksDemo := [][]*item{
		[]*item{
			{"我是老三", 1},
			{"我是老二", 2},
			{"我是老大", 3},
		},
		[]*item{
			{"我是老王", 4},
			{"我是老张", 5},
			{"我是老李", 6},
		},
	}

	ssk.blocks = blocksDemo

	if flag := ssk.Less(1, 0); flag != true {
		t.Error("error")
	}

	if flag := ssk.Less(0, 1); flag != false {
		t.Error("error")
	}
}

// 因为是*item，修改了ssk.blocks也会同样修改了blocksDemo,
//所以就另外用另一个变量存储相同的值，为了后面的比较
func TestSstackSwap(t *testing.T) {
	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	blocksDemo := [][]*item{
		[]*item{
			{"我是老三", 1},
			{"我是老二", 2},
			{"我是老大", 3},
		},
		[]*item{
			{"我是老王", 4},
			{"我是老张", 5},
			{"我是老李", 6},
		},
	}
	compareDemo := [][]*item{
		[]*item{
			{"我是老三", 1},
			{"我是老二", 2},
			{"我是老大", 3},
		},
		[]*item{
			{"我是老王", 4},
			{"我是老张", 5},
			{"我是老李", 6},
		},
	}

	ssk.blocks = blocksDemo

	if ssk.Swap(0, 1); ssk.blocks[0][0].priority != compareDemo[0][1].priority &&
		ssk.blocks[0][0].value != compareDemo[0][1].value {
		t.Error("swap failed")
	}
}

func TestReset(t *testing.T) {
	var (
		setIndex setIndexCallback
		ssk      = newSstack(setIndex)
	)
	ssk.size = 2
	fmt.Println(ssk.Len())
	if ssk.Reset(); ssk.Len() == 2 {
		t.Error("Reset failed")
	}
}
