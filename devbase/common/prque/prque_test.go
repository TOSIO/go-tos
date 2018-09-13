package prque

import (
	"testing"
)

var items = []struct {
	values     interface{}
	priorities int64
}{
	{"我是老三", 1},
	{"我是老二", 2},
	{"我是老大", 3},
}

func TestPush(t *testing.T) {

	var (
		a setIndexCallback
		p = New(a)
	)
	for _, it := range items {
		p.Push(it.values, it.priorities)
	}
	if p.Size() != 3 {
		t.Error("push failed")
	}
}

func TestPop(t *testing.T) {
	var (
		a setIndexCallback
		p = New(a)
	)
	for _, it := range items {
		p.Push(it.values, it.priorities)
	}
	v, p1 := p.Pop()

	if v != items[2].values || p1 != items[2].priorities {
		t.Error("pop error")
	}
}

func TestRemove(t *testing.T) {
	var (
		a setIndexCallback
		p = New(a)
	)
	for _, it := range items {
		p.Push(it.values, it.priorities)
	}

	if p.Remove(2); p.Size() != 2 {
		t.Errorf("删除了一个之后应该是2个，然后所剩下%d个", p.Size())
	}
	if p.Remove(1); p.Size() != 1 {
		t.Errorf("继续删除了一个之后应该是剩下1个了，然而剩下了%d个", p.Size())
	}

}
