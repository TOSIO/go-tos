package statistics

import (
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"
)

type Statistics struct {
	totalCount     *big.Int
	lastTotalCount *big.Int
	lastTime       int64
	lock           sync.RWMutex
}

func (s *Statistics) Init(info string) {
	s.totalCount = big.NewInt(0)
	s.lastTotalCount = big.NewInt(0)
	s.lastTime = time.Now().UnixNano()
	interval := int64(time.Second * 10)
	go func() {
		for {
			nowTime := time.Now().UnixNano()
			s.lock.RLock()
			totalCount := new(big.Int).Set(s.totalCount)
			s.lock.RUnlock()
			if (s.lastTime+interval < nowTime) && (totalCount.Cmp(s.lastTotalCount) > 0) {
				count := new(big.Int).Sub(totalCount, s.lastTotalCount).Int64()
				actualIntervalS := (nowTime - s.lastTime) / int64(time.Second)
				fmt.Println("========================")
				fmt.Println(info)
				fmt.Printf("totalCount = %d\n", totalCount)
				fmt.Printf("lastTotalCount = %d\n", s.lastTotalCount)
				fmt.Printf("%ds totalCount = %d\n", actualIntervalS, count)
				fmt.Printf("%d/s\n", count/actualIntervalS)
				fmt.Println("NumGoroutine:", runtime.NumGoroutine())
				fmt.Println("nowTime:", nowTime/int64(time.Second))
				s.lastTotalCount.Set(totalCount)
				s.lastTime = nowTime
			}
			time.Sleep(time.Millisecond * 100)
		}
	}()
}

func (s *Statistics) Statistics(ok bool) {
	if ok {
		s.lock.Lock()
		s.totalCount = s.totalCount.Add(s.totalCount, big.NewInt(1))
		s.lock.Unlock()
	}
}
