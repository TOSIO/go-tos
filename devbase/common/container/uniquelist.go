package container

import (
	"errors"
	"fmt"
)

type Elem interface{}

type Value struct {
	prev Elem
	next Elem
}

type Iterator struct {
	e    Elem
	cur  *Value
	list *UniqueList
}

type UniqueList struct {
	head      Elem
	tail      Elem
	container map[Elem]*Value
	maxSize   int
}

func NewUniqueList(size int) *UniqueList {
	return &UniqueList{head: nil,
		maxSize:   size,
		tail:      nil,
		container: make(map[Elem]*Value)}
}

func (lm *UniqueList) ContainLen(e Elem) int {
	return len(lm.container)
}

func (lm *UniqueList) IsExist(e Elem) bool {
	_, ok := lm.container[e]
	return ok
}

func (lm *UniqueList) Push(e Elem) error {
	if len(lm.container) >= lm.maxSize {
		return errors.New("exceeded container size limit")
	}
	if lm.IsExist(e) {
		return fmt.Errorf("the element has already exist")
	}
	if lm.head == nil {
		lm.head = e
		lm.tail = e
		val := &Value{nil, nil}
		lm.container[e] = val
	} else {
		val := &Value{lm.tail, nil}
		//lm.container[lm.head].prev = e
		lm.container[lm.tail].next = e
		lm.tail = e
		lm.container[e] = val
	}
	return nil
}

func (lm *UniqueList) Remove(e Elem) {
	if val, ok := lm.container[e]; ok {
		//prevVal, ok := lm.container[val.prev]
		if e == lm.head {
			if len(lm.container) == 1 {
				lm.head = nil
				lm.tail = nil
			} else {
				lm.head = val.next
				if val.next != nil {
					lm.container[val.next].prev = nil
				}
			}
		} else {
			lm.container[val.prev].next = val.next
			if val.next != nil {
				lm.container[val.next].prev = val.prev
			}
			if lm.tail == e {
				lm.tail = val.prev
			}
		}
		delete(lm.container, e)
	}
}

func (lm *UniqueList) Front() (*Iterator, error) {
	if lm.head == nil {
		return nil, fmt.Errorf("list is empty")
	}
	if val, ok := lm.container[lm.head]; ok {
		return &Iterator{lm.head, val, lm}, nil
	} else {
		return nil, fmt.Errorf("list is empty")
	}
}

func (lm *UniqueList) End() (*Iterator, error) {
	if lm.tail == nil {
		return nil, fmt.Errorf("list is empty")
	}
	if val, ok := lm.container[lm.tail]; ok {
		return &Iterator{lm.tail, val, lm}, nil
	} else {
		return nil, fmt.Errorf("list is empty")
	}
}
func (itr *Iterator) HasNext() bool {
	if itr.cur != nil {
		return itr.list.container[itr.cur].next == nil
	} else {
		return false
	}
}

func (itr *Iterator) Next() *Iterator {
	if itr.cur != nil && itr.cur.next != nil {
		next, _ := itr.list.container[itr.cur.next]
		return &Iterator{itr.cur.next, next, itr.list}
	}
	return nil
}

func (itr *Iterator) Data() Elem {
	return itr.e
}

func (itr *Iterator) Equal(other *Iterator) bool {
	return other != nil && itr == other || itr.Data() == other.Data()
}
