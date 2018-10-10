package statistics

import (
	"fmt"
	"math/big"
	"time"
)

type Statistics struct {
	totalCount     *big.Int
	lastTotalCount *big.Int
	lastTime       int64
}

func (s *Statistics) Statistics() {
	if s.totalCount == nil {
		s.totalCount = big.NewInt(0)
		s.lastTotalCount = big.NewInt(0)
		s.lastTime = time.Now().Unix()
	}
	s.totalCount = s.totalCount.Add(s.totalCount, big.NewInt(1))
	nowTime := time.Now().Unix()
	if s.lastTime+10 < nowTime {
		temp := big.NewInt(0)
		temp = temp.Sub(s.totalCount, s.lastTotalCount)
		tempInt := temp.Int64()
		fmt.Println("========================")
		fmt.Println("totalCount=", s.totalCount)
		fmt.Println("lastTotalCount=", s.lastTotalCount)
		fmt.Println("10 s totalCount=", tempInt)
		fmt.Println(tempInt/10, "/s")
		s.lastTotalCount.Set(s.totalCount)
		s.lastTime = nowTime
	}
}
