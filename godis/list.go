package godis

import (
	"log"
)

type Node struct {
	prev *Node
	next *Node
	val  *GodisObj
}

type GodisList struct {
	head   *Node
	tail   *Node
	length uint64
}

func ListCreate() GodisList {
	return GodisList{
		head:   nil,
		tail:   nil,
		length: 0,
	}
}

func (li *GodisList) listLength() uint64 {
	return li.length
}

func (li *GodisList) listFirst() *Node {
	return li.head
}

func (li *GodisList) listLast() *Node {
	return li.tail
}

func (li *GodisList) listPrevNode(node *Node) *Node {
	if node != nil {
		return node.prev
	}
	return nil
}

func (li *GodisList) listNextNode(node *Node) *Node {
	if node != nil {
		return node.next
	}
	return nil
}

func (li *GodisList) listAddNodeHead(obj *GodisObj) {
	node := Node{
		prev: nil,
		next: nil,
		val:  obj,
	}
	if li.head == nil {
		li.head = &node
		li.tail = &node
	} else {
		node.next = li.head
		li.head.prev = &node
		li.head = &node
	}
	li.length++
}

func (li *GodisList) listPopNodeHead() *Node {
	if li.head != nil {
		tmp := li.head
		li.head = li.head.next
		if li.head != nil {
			li.head.prev = nil
		} else {
			li.tail = nil
		}
		tmp.next = nil

		li.length--
		return tmp
	} else {
		return nil
	}

}

func (li *GodisList) listAddNodeTail(obj *GodisObj) {
	node := Node{
		prev: nil,
		next: nil,
		val:  obj,
	}
	if li.tail == nil {
		li.head = &node
		li.tail = &node
	} else {
		node.prev = li.tail
		li.tail.next = &node
		li.tail = &node
	}
	li.length++
}

func (li *GodisList) listPopNodeTail() *Node {

	if li.tail != nil {
		tmp := li.tail
		li.tail = li.tail.prev
		if li.tail != nil {
			li.tail.next = nil
		} else {
			li.head = nil
		}
		li.length--
		tmp.prev = nil
		return tmp
	} else {
		return nil
	}

}

func indexConvert(n int64, length uint64) uint64 {
	if n >= 0 {
		return uint64(n)
	}
	var tmp int64 = n + int64(length)
	return uint64(tmp)
}

func (li *GodisList) listGetIndex(index int64) *Node {
	if index < 0 {
		index = -index
		if index > int64(li.length) {
			return nil
		}
		index = -index
		index = int64(li.length) + index
	}
	if index >= int64(li.length) {
		return nil
	}
	if index <= int64(li.length)/2 {
		node := li.head
		remain := index
		for remain > 0 {
			remain--
			node = node.next
		}
		return node
	} else {
		node := li.tail
		remain := li.length - uint64(index) - 1
		for remain > 0 {
			remain--
			node = node.prev
		}
		return node
	}

}

func (li *GodisList) listSetIndex(index uint64, obj *GodisObj) {
	if index < 0 || index >= li.length {
		return
	}
	node := li.head
	remain := index
	for remain > 0 {
		remain--
		node = node.next
	}
	node.val = obj
}

func (li *GodisList) listSearchFirst(obj *GodisObj) *Node {
	node := li.head
	for node != nil {
		if strValNode, ok := node.val.val.(string); ok {
			if strValObj, ok := obj.val.(string); ok {
				if strValNode == strValObj {
					return node
				} else {
					node = node.next
				}
			} else {
				log.Println("error when listSearch")
				return nil
			}
		} else {
			log.Println("error when listSearch")
			return nil
		}
	}
	return nil
}

func (li *GodisList) listDelNode(node *Node) {
	if node.prev == nil {
		li.listPopNodeHead()
	} else if node.next == nil {
		li.listPopNodeTail()
	} else {
		node.prev.next = node.next
		node.next.prev = node.prev
		node.next = nil
		node.prev = nil
		li.length--
	}
}

func (li *GodisList) listTestIndex(index int) bool {
	if index < 0 {
		index = -index
		if index > int(li.length) {
			return false
		}
		index = -index
	}
	if index >= int(li.length) {
		return false
	}
	return true
}

func (li *GodisList) listAbsIndex(index int) int {
	if index < 0 {
		index = -index
		if index > int(li.length) {
			return 0
		}
		index = -index
		index = int(li.length) + index
	}
	if index >= int(li.length) {
		return int(li.length) - 1
	}
	return index
}
