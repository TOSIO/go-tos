package sdag

import (
	"testing"
	"github.com/TOSIO/go-tos/sdag/manager"
	"fmt"
	"github.com/TOSIO/go-tos/devbase/utils"
)

func TestQUeryBlockInfo(t *testing.T){

    var Time  uint64
	Time = 1540180800000
	Time = utils.GetMainTime(Time)
	
	var queryBlock *manager.QueryBlockInfo
	mainBlockInfo := queryBlock.GetMainBlockInfo(Time)
	fmt.Println(mainBlockInfo)
}