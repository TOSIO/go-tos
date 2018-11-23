package manager_test

import (
	"container/list"
	"fmt"
	"testing"
	"time"

	"github.com/TOSIO/go-tos/devbase/statistics"
)

func TestList(t *testing.T) {
	var statisticsObj statistics.Statistics
	quit := make(chan bool)
	testList := list.New()
	go func() {
		for i := 0; i < 1000000000000; i++ {
			testList.PushFront(i)
			statisticsObj.Statistics()
			//time.Sleep(10 * time.Millisecond)
		}
		quit <- true
	}()
	for j := 0; j < 4; j++ {
		go func(index int) {
			for {
				i := 0
				for e := testList.Front(); e != nil; e = e.Next() {
					i++
					//fmt.Printf("e : %d ", e.Value)
					if i > 4 {
						break
					}
				}
				fmt.Printf("====================%d\n", index)
				time.Sleep(50 * time.Millisecond)

			}
		}(j)
	}

	<-quit
}
