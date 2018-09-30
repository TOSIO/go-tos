package storage

import (
	"testing"
	"github.com/TOSIO/go-tos/devbase/utils"
	"fmt"
	//"github.com/TOSIO/go-tos/devbase/common/hexutil"
	"time"
)

func TestBlockLookUpKey(t *testing.T) {

	for i := 0; i < 100 ; i++ {
		tm := utils.GetTimeStamp()
		//fmt.Println(hexutil.Encode(blockLookUpKey(utils.GetMainTime(tm))))
		fmt.Println(blockLookUpKey(utils.GetMainTime(tm)))

		time.Sleep(1 * time.Millisecond)
	}
}