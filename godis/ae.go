package godis

import (
	"log"
	"time"

	"golang.org/x/sys/unix"
)

type AeState int

type AeErr struct {
	msg string `default:"AE_ERR"`
}

func (e AeErr) Error() string {
	return e.msg
}

func (e *AeErr) Is(tgt error) bool {
	switch tgt.(type) {
	case *AeErr, AeErr:
		return true
	}
	return false
}

func (e *AeErr) As(target interface{}) bool {
	switch v := target.(type) {
	case *AeErr:
		v.msg = e.msg
		return true
	}
	return false
}

type FileEventType int

const (
	AE_NONE     FileEventType = 0x0
	AE_READABLE FileEventType = 0x1
	AE_WRITABLE FileEventType = 0x2
)

type TimeEventType int

const (
	AE_NORMAL TimeEventType = 0x1
	AE_ONCE   TimeEventType = 0x2
)

type FileProc func(loop *AeEventLoop, fd int, mask FileEventType, extra interface{})
type TimeProc func(loop *AeEventLoop, fd int, extra interface{}) int

type AeFileEvent struct {
	fd         int
	mask       FileEventType /* one of AE_(READABLE|WRITABLE) */
	write_proc FileProc
	read_proc  FileProc
	extra      interface{}
}

type AeTimeEvent struct {
	id    int
	when  int //ms
	mask  TimeEventType
	proc  TimeProc
	next  *AeTimeEvent
	extra interface{}
}

type EpollState struct {
	epfd            int
	events          []unix.EpollEvent
	readyEventCount int
	regfd           map[int]FileEventType
}

type AeEventLoop struct {
	fileEvents map[int]*AeFileEvent

	timeEventNextId int
	timeEventHead   *AeTimeEvent
	lastTime        int
	stop            bool
	epoll           *EpollState
}

func AeCreateEventLoop() (*AeEventLoop, error) {
	epoll_fd, err := unix.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	epoll_state := &EpollState{
		epfd:            epoll_fd,
		events:          make([]unix.EpollEvent, 1024),
		readyEventCount: 0,
		regfd:           make(map[int]FileEventType),
	}

	loop := AeEventLoop{
		fileEvents:      make(map[int]*AeFileEvent),
		timeEventNextId: 1,
		timeEventHead:   nil,
		lastTime:        GetMsTime(),
		stop:            false,
		epoll:           epoll_state,
	}

	return &loop, err
}

func (loop *AeEventLoop) AeStop() {
	loop.stop = true
}

func (loop *AeEventLoop) AeCreateFileEvent(fd int, mask FileEventType, proc FileProc, extra interface{}) error {
	_, ok := loop.fileEvents[fd]
	if !ok {
		loop.fileEvents[fd] = &AeFileEvent{
			fd:         fd,
			mask:       AE_NONE,
			write_proc: nil,
			read_proc:  nil,
			extra:      extra,
		}
	}

	ev := loop.fileEvents[fd]
	ev.mask |= AE_READABLE
	if mask&AE_READABLE == AE_READABLE {
		ev.read_proc = proc
	}
	if mask&AE_WRITABLE == AE_WRITABLE {
		ev.write_proc = proc
	}

	err := loop.aeEpollAdd(fd, mask)
	if err != nil {
		delete(loop.fileEvents, fd)
		return err
	}
	return nil

}

func (loop *AeEventLoop) aeEpollAdd(fd int, mask FileEventType) error {
	regop, ok := loop.epoll.regfd[fd]
	var op int = 0
	if ok && regop != 0 {
		loop.epoll.regfd[fd] |= mask
		op = unix.EPOLL_CTL_MOD
	} else {
		loop.epoll.regfd[fd] = mask
		op = unix.EPOLL_CTL_ADD
	}
	var ev uint32 = 0
	if regop&AE_READABLE == AE_READABLE {
		ev |= unix.EPOLLIN
	}
	if regop&AE_WRITABLE == AE_WRITABLE {
		ev |= unix.EPOLLOUT
	}

	if mask&AE_READABLE == AE_READABLE {
		ev |= unix.EPOLLIN
	}
	if mask&AE_WRITABLE == AE_WRITABLE {
		ev |= unix.EPOLLOUT
	}
	err := unix.EpollCtl(loop.epoll.epfd, op, fd, &unix.EpollEvent{
		Fd:     int32(fd),
		Events: ev,
	})
	if err != nil {
		if ok {
			loop.epoll.regfd[fd] = regop
		} else {
			delete(loop.epoll.regfd, fd)
		}

		return err
	}
	return nil
}

func (loop *AeEventLoop) AeDeleteFileEvent(fd int, mask FileEventType, extra interface{}) error {
	_, ok := loop.fileEvents[fd]
	if !ok {
		return nil
	}
	loop.fileEvents[fd].mask &^= mask

	err := loop.aeEpollDelete(fd, mask)
	if err != nil {
		return err
	}
	if loop.fileEvents[fd].mask == AE_NONE {
		delete(loop.fileEvents, fd)
	}
	return nil
}

func (loop *AeEventLoop) aeEpollDelete(fd int, mask FileEventType) error {
	regop, ok := loop.epoll.regfd[fd]
	var ev uint32 = 0
	if !ok {
		return nil
	}
	if regop&AE_READABLE == AE_READABLE {
		ev |= unix.EPOLLIN
	}
	if regop&AE_WRITABLE == AE_WRITABLE {
		ev |= unix.EPOLLOUT
	}
	if mask&AE_READABLE == AE_READABLE {
		ev &^= unix.EPOLLIN
	}
	if mask&AE_WRITABLE == AE_WRITABLE {
		ev &^= unix.EPOLLOUT
	}

	if ev == 0 {
		err := unix.EpollCtl(loop.epoll.epfd, unix.EPOLL_CTL_DEL, fd, nil)
		if err != nil {
			return err
		}
		delete(loop.epoll.regfd, fd)
	} else {
		err := unix.EpollCtl(loop.epoll.epfd, unix.EPOLL_CTL_MOD, fd, &unix.EpollEvent{
			Fd:     int32(fd),
			Events: ev,
		})
		if err != nil {
			return err
		}
		loop.epoll.regfd[fd] &^= mask
	}
	return nil
}

func (loop *AeEventLoop) AeCreateTimeEvent(milliseconds int, mask TimeEventType, proc TimeProc, extra interface{}) int {
	var id int = loop.timeEventNextId
	loop.timeEventNextId++
	te := &AeTimeEvent{
		id:    id,
		when:  milliseconds,
		mask:  mask,
		proc:  proc,
		next:  nil,
		extra: extra,
	}
	te.next = loop.timeEventHead
	loop.timeEventHead = te

	return id
}

func (loop *AeEventLoop) AeDeleteTimeEvent(id int) error {
	var te, prev *AeTimeEvent = nil, nil
	te = loop.timeEventHead
	for te != nil {
		if te.id == id {
			if prev == nil {
				loop.timeEventHead = te.next
			} else {
				prev.next = te.next
			}
			return nil
		}
		prev = te
		te = te.next
	}
	return AeErr{}
}

func (loop *AeEventLoop) aeProcessEvents() uint64 {
	var processed uint64 = 0
	if loop.epoll.readyEventCount > 0 {
		for i := 0; i < loop.epoll.readyEventCount; i++ {
			fd := loop.epoll.events[i].Fd
			ev := loop.epoll.events[i].Events
			//var repoll int = 0
			if ev&unix.EPOLLIN == unix.EPOLLIN {
				loop.fileEvents[int(fd)].read_proc(loop, int(fd), AE_READABLE, nil)
				//repoll = 1
			}

			if ev&unix.EPOLLOUT == unix.EPOLLOUT {
				loop.fileEvents[int(fd)].write_proc(loop, int(fd), AE_WRITABLE, nil)
			}
		}
		processed++
	}

	//timeEvent process

	now := GetMsTime()
	//handle time skew
	if now < loop.lastTime {
		te := loop.timeEventHead
		for te != nil {
			te.when = 0
			te = te.next
		}
	}

	te := loop.timeEventHead
	for te != nil {
		now = GetMsTime()
		if now > te.when {
			re_exec := te.proc(loop, te.id, nil)
			processed++

			if te.mask&AE_NORMAL == AE_NORMAL {
				te.when = GetMsTime() + re_exec
			} else {
				loop.AeDeleteTimeEvent(te.id)
			}
			te = loop.timeEventHead
		} else {
			te = te.next
		}

	}

	return processed
}

func (loop *AeEventLoop) aeSearchNearestTimer() *AeTimeEvent {
	var te, nearest *AeTimeEvent = loop.timeEventHead, nil
	for te != nil {
		if nearest == nil || te.when < nearest.when {
			nearest = te
		}
		te = te.next
	}
	return nearest
}

func GetMsTime() int {
	return int(time.Now().UnixNano() / 1e6)
}

func calTimeInterval(loop *AeEventLoop) int {
	now := GetMsTime()
	shortest := loop.aeSearchNearestTimer()
	var wait_time int = 10 // time epoll block
	if shortest != nil {
		time_interval := shortest.when - now
		if time_interval < 0 {
			wait_time = 0
		} else {
			wait_time = time_interval
		}
	}
	return wait_time
}

func (loop *AeEventLoop) aeWait() {
	wait_time := calTimeInterval(loop)
	n, err := unix.EpollWait(loop.epoll.epfd, loop.epoll.events, wait_time)
	if err != nil && err != unix.EINTR {
		log.Panicf("EpollWait: %v\n", err)
	}
	loop.epoll.readyEventCount = n

}

func (loop *AeEventLoop) AeMain() {
	loop.stop = false
	for !loop.stop {
		loop.aeWait()
		loop.aeProcessEvents()
	}
}
