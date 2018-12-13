package statistics

import (
	"fmt"
	"github.com/TOSIO/go-tos/devbase/log"
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
				str := fmt.Sprintln("\n=========================")
				str += fmt.Sprintln(info)
				str += fmt.Sprintf("totalCount = %d\n", totalCount)
				str += fmt.Sprintf("lastTotalCount = %d\n", s.lastTotalCount)
				str += fmt.Sprintf("%ds totalCount = %d\n", actualIntervalS, count)
				str += fmt.Sprintf("%d/s\n", count/actualIntervalS)
				str += fmt.Sprintln("NumGoroutine:", runtime.NumGoroutine())
				str += fmt.Sprintln("nowTime:", nowTime/int64(time.Second))
				str += fmt.Sprintf("=========================")
				log.Warn(str)
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
