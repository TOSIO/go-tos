package utils

import (
	"time"
)

/*
1. 获取当前时间戳 毫秒
2. 根据时间戳计算时间片位置
3.

 */
func GetTimeStamp() int64 {
	return time.Now().UnixNano() / 1e6
}
