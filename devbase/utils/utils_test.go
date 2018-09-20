package utils

import (
	"testing"
	"time"
	"fmt"
)

func TestGetTOSTimeStamp(t *testing.T) {
	tm := time.Now().UnixNano() / 1e6

	fmt.Println(tm)
	//1537435886798

	for i := 1; i < 100; i++ {
		tosTime := GetTOSTimeStamp()
		fmt.Println(tosTime)
		time.Sleep(1 * time.Millisecond)
	}

}
