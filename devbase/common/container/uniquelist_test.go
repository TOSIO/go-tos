package container_test

import (
	"fmt"
	"testing"

	"github.com/TOSIO/go-tos/devbase/common/container"
	//"github.com/TOSIO/go-tos/devbase/common/container"
)

func TestLinkedList(t *testing.T) {
	list := container.NewUniqueList()

	err := list.Push(12)
	if err != nil {
		return
	}
	err = list.Push(56)
	err = list.Push(45)
	err = list.Push(78)
	err = list.Push(100)

	//itrEnd, err := list.End()
	for itr, _ := list.Front(); itr != nil; itr = itr.Next() {
		fmt.Printf("e:%d ", itr.Data())
	}

	fmt.Println("\n=============================")
	list.Remove(12)
	list.Push(22)
	list.Push(23)
	for itr, _ := list.Front(); itr != nil; itr = itr.Next() {
		fmt.Printf("e:%d ", itr.Data())
	}

	fmt.Println("\n=============================")
	list.Remove(78)
	for itr, _ := list.Front(); itr != nil; itr = itr.Next() {
		fmt.Printf("e:%d ", itr.Data())
	}

	fmt.Println("\n=============================")
	list.Remove(100)
	for itr, _ := list.Front(); itr != nil; itr = itr.Next() {
		fmt.Printf("e:%d ", itr.Data())
	}

	return
}
