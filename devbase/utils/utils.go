package utils

import (
	"time"
	"github.com/TOSIO/go-tos/params"
)

/*
1. 获取当前时间戳 毫秒
2. 根据时间戳计算时间片位置
3.
 */
func GetTOSTimeStamp() uint64 {
	t := uint64(time.Now().UnixNano()) / 1e6
	//fmt.Println(t)
	mTime := t / params.TimePeriod
	sTime := t % params.TimePeriod

	return mTime*1e5 + sTime
}

func TOSTimeStampToTime(tosT uint64) uint64 {
	mTime  := tosT / 1e5
	sTime  := tosT % 1e5

	return mTime * params.TimePeriod + sTime
}

func GetTimeStamp() uint64 {
	return uint64(time.Now().UnixNano()) / 1e6
}

func GetMainTime(timeStamp uint64) uint64 {

	return timeStamp / params.TimePeriod
}