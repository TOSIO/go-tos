package mclock

import (
	"testing"
	"time"

	"github.com/aristanetworks/goarista/monotime"
)

func TestNow(t *testing.T) {
	if Now() != AbsTime(monotime.Now()) {
		t.Error("should be equal")
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		current  AbsTime
		duration time.Duration
		want     AbsTime
	}{
		{AbsTime(20180912), time.Duration(2), AbsTime(20180914)},
		{AbsTime(Now()), time.Duration(2), AbsTime(Now() + 2)},
	}

	for _, test := range tests {
		if result := test.current; result.Add(test.duration) != test.want {
			t.Errorf("actual:%v,want:%v", result, test.want)
		}
	}
}
