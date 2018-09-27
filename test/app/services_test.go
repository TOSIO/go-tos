package services_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/TOSIO/go-tos/app/sendTx/httpSend"
)

/*
func TestClientsTPS(t *testing.T) { //单线程

	var (
		send = `
		{
			"jsonrpc":"2.0",
			"method":"sdag_transaction",
			"params":["{\"Form\":{\"Address\" :\"0x0f56913b0468147732b83FC465C4c87D110a4804\",\"PublicKey\" :\"04de0ae651b05dfa5b9ef4e30d533a8b8f1e776fa3f611708c35e0a5b11b73fe2da7c137b7fc79ba7327f628d258e4cd1197f794ed6fcbb72729899f276ead9444\",\"PrivateKey\"  :\"a3386b30fe10f50412c03f4af7d12085aa0cd1901d3ff0df8ed2931d97f056db\"},\"To\":\"0x9C3B103Dc4B7d1FF2CA756904f6981655013A5c9\",\"Amount\":\"1049938402741431209448\"}"],
			"id":1
		}`
		urlString = "http://192.168.1.21:8545"
	)
	var failCount int64 = 0
	var totalTime int64 = 0
	//var timeNumber int = 1e9
	var totalPackage int64 = 100
	for i := 0; i < 100; i++ {
		beginTime := time.Now().UnixNano()
		recives, err := httpSend.SendHttp(urlString, send)
		endTime := time.Now().UnixNano()
		fmt.Printf("%s", recives)
		if err != nil {
			failCount++
		}
		totalTime += (endTime - beginTime)
	}
	fmt.Printf("%d\n", totalTime)
	realPackageCount := totalPackage - failCount
	TPS := realPackageCount * 1e9 / totalTime
	fmt.Printf("%v", TPS)

}
*/
/* func maxTotalTime(totalTime []float64) (maxTotalTime float64) {

	for j := 0; j < len(totalTime)-1; j++ {
		for i := 1; i < len(totalTime); i++ {
			totalTime[i] = math.Max(totalTime[j], totalTime[i])
		}
	}
	maxTotalTime = totalTime[len(totalTime)]
	return
} */
func TestClientTPS(t *testing.T) { //多线程
	/* var totalPackage = []int64{
		1000, 8000, 10000, 20000, 50000, 100000,
	} */
	var totalPackage int64 = 10000
	const threadCount int64 = 1000
	time := make(chan float64)
	count := make(chan int64)
	done := make(chan bool)
	var realTotalTime float64 = 0
	var failCount int64
	var totalTime [threadCount]float64
	//var timeNumber int64 = 1e9
	//for j := 0; j < len(totalPackage); j++ {

	failCount = 0
	var i int64
	for i = 0; i < threadCount; i++ {
		go thread(time, count, totalPackage/threadCount)
	}

	go func() {
		var q int64

		for q = 0; q < threadCount; q++ {
			totalTime[q] = <-time
			if totalTime[q] > realTotalTime {
				realTotalTime = totalTime[q]
			}
			failCount += <-count
		}
		done <- true
	}()
	<-done
	/* 	for j := 0; j < len(totalTime)-1; j++ {
		for i := 1; i < len(totalTime); i++ {
			totalTime[i] = math.Max(totalTime[j], totalTime[i])
			realTotalTime = totalTime[i]
		}
	} */
	fmt.Printf("second:%.8f\n", realTotalTime)
	realPackageCount := totalPackage - failCount
	realPackageCounts := strconv.FormatInt(realPackageCount, 10)
	floatPackageCounts, _ := strconv.ParseFloat(realPackageCounts, 64)
	TPS := floatPackageCounts / realTotalTime
	fmt.Printf("totalPackage:%d,failCount :%d,realPackageCount:%d,TPS：%.0f\n", totalPackage, failCount, realPackageCount, TPS)

	/* stotalTime := strconv.FormatInt(totalTime, 10)
	floatTotalTime, _ := strconv.ParseFloat(stotalTime, 64)
	fmt.Printf("%v\n", floatTotalTime/1e9)
	realPackageCount := totalPackage[j] - failCount
	realPackageCounts := strconv.FormatInt(realPackageCount, 10)
	floatPackageCounts, _ := strconv.ParseFloat(realPackageCounts, 64)
	TPS := floatPackageCounts * 1e9 / floatTotalTime
	fmt.Printf("totalCount:%v,failCount :%v,TPS：%v\n", totalPackage[j], failCount, TPS)
	*/

}

func thread(times chan float64, counts chan int64, threadCounts int64) {

	var (
		send = `
		{
			"jsonrpc":"2.0",
			"method":"sdag_transaction",
			"params":["{\"Form\":{\"Address\" :\"0x0f56913b0468147732b83FC465C4c87D110a4804\",\"PublicKey\" :\"04de0ae651b05dfa5b9ef4e30d533a8b8f1e776fa3f611708c35e0a5b11b73fe2da7c137b7fc79ba7327f628d258e4cd1197f794ed6fcbb72729899f276ead9444\",\"PrivateKey\"  :\"a3386b30fe10f50412c03f4af7d12085aa0cd1901d3ff0df8ed2931d97f056db\"},\"To\":\"0x9C3B103Dc4B7d1FF2CA756904f6981655013A5c9\",\"Amount\":\"1049938402741431209448\"}"],
			"id":1
		}`
		urlString = "http://192.168.1.21:8545"
	)
	var failCount int64 = 0
	//var totalTime int64 = 0
	var totalTime time.Duration
	var i int64
	for i = 0; i < threadCounts; i++ {
		beginTime := time.Now().UTC()
		_, err := httpSend.SendHttp(urlString, send)
		endTime := time.Now().UTC()
		if err != nil {
			failCount++
		}
		//fmt.Printf("%s", recives)
		totalTime += (endTime.Sub(beginTime))
	}

	times <- totalTime.Seconds()
	counts <- failCount

}

/*
func TestClientThreadCounts(t *testing.T) {
	var totalPackage int64 = 10000
	var threadCount = []int64{
		1000, 700, 600, 500, 400, 300, 200, 100, 10,
	}
	//const threadCount int64 = 1000
	time := make(chan float64)
	count := make(chan int64)
	done := make(chan bool)
	var realTotalTime float64
	var failCount int64

	for j := 0; j < len(threadCount); j++ {
		failCount = 0

		var totalTime []float64
		var i int64
		for i = 0; i < threadCount[j]; i++ {
			//runtime.GOMAXPROCS()
			go thread(time, count, totalPackage/threadCount[j])
		}
		go func() {
			var q int64
			for q = 0; q < threadCount[j]; q++ {
				totalTime[q] = <-time
				failCount += <-count
			}
			done <- true
		}()
		<-done
		for j := 0; j < len(totalTime)-1; j++ {
			for i := 1; i < len(totalTime); i++ {
				totalTime[i] = math.Max(totalTime[j], totalTime[i])
				realTotalTime = totalTime[i]
			}
		}
		fmt.Printf("%.8f\n", realTotalTime)
		//	realPackageCount := totalPackage - failCount
		//	TPS := realPackageCount * 1e9 / totalTime
		//	fmt.Printf("totalCount:%v,failCount :%v,TPS：%v\n", threadCount[j], failCount, TPS)

		/* 	stotalTime := strconv.FormatInt(totalTime, 10)
		floatTotalTime, _ := strconv.ParseFloat(stotalTime, 64)
		realPackageCount := totalPackage - failCount
		realPackageCounts := strconv.FormatInt(realPackageCount, 10)
		floatPackageCounts, _ := strconv.ParseFloat(realPackageCounts, 64)
		TPS := floatPackageCounts * 1e9 / floatTotalTime
		fmt.Printf("totalCount:%v,failCount :%v,TPS：%v\n", threadCount[j], failCount, TPS)
	}
}*/
/*
func TestServeTPS(t *testing.T) {

} */
