package utils

import (
	"time"
)

/*
1. 获取当前时间戳 毫秒
2. 根据时间戳计算时间片位置
3.
 */
func GetTOSTimeStamp() uint64 {
	t := uint64(time.Now().UnixNano()) / 1e6
	//fmt.Println(t)
	mTime := t >> 16
	sTime := t & (1<<16 - 1)

	return mTime*1e5 + sTime
}


func TOSTimeStampToTime(tosT uint64) uint64 {
	mTime  := tosT / 1e5
	sTime  := tosT % 1e5

	return (mTime << 16) | sTime
}
