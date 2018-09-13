package main

import (
	"fmt"

	"github.com/TOSIO/go-tos/devbase/common/prque"
)

func foo(a interface{}, i int) {
	fmt.Println("call foo")
}

func main() {
	vPrque := prque.New(foo)
	vPrque.Push(1, 1)
	vPrque.Push(3, 0)
	vPrque.Push(2, 1)
	vPrque.Push(4, 2)
	vPrque.Remove(3)
	for !vPrque.Empty() {
		fmt.Println(vPrque.Pop())
	}
}
