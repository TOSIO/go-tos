package mclock

import (
	"fmt"
	"testing"
)

func TestRun(t *testing.T) {
	events := []struct {
		does      func() int
		atOneTime AbsTime
	}{
		{does: func() int {
			fmt.Println("这是第一条")
			return 1
		}, atOneTime: Now()},
		{does: func() int {
			fmt.Println("这是第二条")
			return 1
		}, atOneTime: Now() + AbsTime(1)},
	}

	for _, e := range events {
		var s Simulated
		if s.Run(2); e.does() != 1 {
			t.Error("the before Timers do not execute")
		}
	}
}
