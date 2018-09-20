package utils

import (
	"time"
	"fmt"
)

/*
1. 获取当前时间戳 毫秒
2. 根据时间戳计算时间片位置
3.

 */
func GetTOSTimeStamp() int64 {
	t := time.Now().UnixNano() / 1e6
	//fmt.Println(t)
	mTime := t >> 16
	sTime := t & (1 << 16 - 1)

	return mTime * 1e5 + sTime
}


