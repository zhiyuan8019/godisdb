package godis

import (
	"math/rand"
	"time"
)

const maxLevel int = 32
const probability float32 = 0.25

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

type splistLevel struct {
	span    int
	forward *splistNode
}

type splistNode struct {
	level []splistLevel

	backward *splistNode
	score    float64
	obj      *GodisObj
}

type GodisSkipList struct {
	head   *splistNode
	tail   *splistNode
	length uint64
	level  int
}

func CreateSkipList() *GodisSkipList {
	h := &splistNode{
		level: make([]splistLevel, maxLevel),
	}
	return &GodisSkipList{
		head:   h,
		tail:   nil,
		length: 0,
		level:  0,
	}
}

func (sl *GodisSkipList) randomLevel() int {
	level := 1
	for float32(r.Float32()) < probability && level < maxLevel {
		level++
	}
	return level
}

func (sl *GodisSkipList) spNext(node *splistNode) *splistNode {
	if node == nil {
		return nil
	}
	return node.level[0].forward

}

func (sl *GodisSkipList) spInsert(score float64, obj *GodisObj) {
	update := make([]*splistNode, maxLevel)
	rank := make([]int, maxLevel)
	x := sl.head
	for i := sl.level - 1; i >= 0; i-- {
		if i == sl.level-1 {
			rank[i] = 0
		} else {
			rank[i] = rank[i+1]
		}
		for x.level[i].forward != nil && (x.level[i].forward.score < score ||
			(x.level[i].forward.score == score && !CompareStrObj(x.level[i].forward.obj, obj))) {
			rank[i] += x.level[i].span
			x = x.level[i].forward
		}

		update[i] = x
	}
	/*from t_zset.c
	* we assume the key is not already inside, since we allow duplicated
	* scores, and the re-insertion of score and redis object should never
	* happen since the caller of zslInsert() should test in the hash table
	* if the element is already inside or not.
	 */
	level := sl.randomLevel()
	if level > sl.level {
		for i := sl.level; i < level; i++ {
			rank[i] = 0
			update[i] = sl.head
			update[i].level[i].span = int(sl.length)
		}
		sl.level = level
	}

	node := &splistNode{
		level: make([]splistLevel, level),
		score: score,
		obj:   obj,
	}
	for i := 0; i < level; i++ {
		node.level[i].forward = update[i].level[i].forward
		update[i].level[i].forward = node

		node.level[i].span = update[i].level[i].span - (rank[0] - rank[i])
		update[i].level[i].span = (rank[0] - rank[i]) + 1
	}

	for i := level; i < sl.level; i++ {
		update[i].level[i].span++
	}

	node.backward = nil
	if update[0] != sl.head {
		node.backward = update[0]
	}
	if node.level[0].forward != nil {
		node.level[0].forward.backward = node
	} else {
		sl.tail = x
	}

	sl.length++
}

func (sl *GodisSkipList) spDelete(score float64, obj *GodisObj) {
	update := make([]*splistNode, maxLevel)
	x := sl.head
	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && (x.level[i].forward.score < score ||
			(x.level[i].forward.score == score && !CompareStrObj(x.level[i].forward.obj, obj))) {
			x = x.level[i].forward
		}
		update[i] = x
	}

	x = x.level[0].forward
	if x.score == score && CompareStrObj(x.obj, obj) {
		sl.deleteNode(x, update)
	}
}

func (sl *GodisSkipList) deleteNode(x *splistNode, update []*splistNode) {
	for i := 0; i < sl.level; i++ {
		if update[i].level[i].forward == x {
			update[i].level[i].span += x.level[i].span - 1
			update[i].level[i].forward = x.level[i].forward
		} else {
			update[i].level[i].span--
		}
	}
	if x.level[0].forward != nil {
		x.level[0].forward.backward = x.backward
	} else {
		sl.tail = x.backward
	}
	for sl.level > 1 && sl.head.level[sl.level-1].forward == nil {
		sl.level--
	}
	sl.length--
}

func (sl *GodisSkipList) spGetRank(score float64, obj *GodisObj) int {
	rank := 0
	x := sl.head
	for i := sl.level - 1; i >= 0; i-- {

		for x.level[i].forward != nil && (x.level[i].forward.score < score ||
			(x.level[i].forward.score == score && !CompareStrObj(x.level[i].forward.obj, obj))) {
			rank += x.level[i].span
			x = x.level[i].forward
		}

		if x.level[i].forward != nil && x.level[i].forward.score == score && CompareStrObj(x.level[i].forward.obj, obj) {
			rank += x.level[i].span
			return rank
		}

	}
	return 0
}

func (sl *GodisSkipList) spGetRangeCount(start float64, stop float64) int {
	rank_start := 0
	x := sl.head
	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && x.level[i].forward.score < start {
			rank_start += x.level[i].span
			x = x.level[i].forward
		}
	}
	rank_stop := 0
	y := sl.head
	for i := sl.level - 1; i >= 0; i-- {
		for y.level[i].forward != nil && y.level[i].forward.score <= stop {
			rank_stop += y.level[i].span
			y = y.level[i].forward
		}
	}
	return rank_stop - rank_start
}

func (sl *GodisSkipList) spFirstInRange(start float64, stop float64) *splistNode {
	x := sl.head
	for i := sl.level - 1; i >= 0; i-- {
		for x.level[i].forward != nil && x.level[i].forward.score < start {
			x = x.level[i].forward
		}
	}
	if x.level[0].forward == nil && x.score < start {
		return nil
	} else {
		return x.level[0].forward
	}
}
