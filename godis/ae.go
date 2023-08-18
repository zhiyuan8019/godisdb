package godis

import (
	"log"

	"golang.org/x/sys/unix"
)

type FileEventType int

const (
	AE_NONE     FileEventType = 0x0
	AE_READABLE FileEventType = 0x1
	AE_WRITABLE FileEventType = 0x2
)

type TimeEventType int

const (
	AE_NORMAL TimeEventType = iota
	AE_ONCE
)

type FileProc func(loop *AeEventLoop, fd int, mask FileEventType, extra interface{})
type TimeProc func(loop *AeEventLoop, fd int, extra interface{})

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
	}

	loop := AeEventLoop{
		timeEventNextId: 0,
		timeEventHead:   nil,
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

	err := loop.aeEpollDelete(fd)
	if err != nil {
		return err
	}
	delete(loop.fileEvents, fd)
	return nil
}

func (loop *AeEventLoop) aeEpollDelete(fd int) error {
	regop, ok := loop.epoll.regfd[fd]
	if ok && regop != 0 {
		err := unix.EpollCtl(loop.epoll.epfd, unix.EPOLL_CTL_DEL, fd, nil)
		if err != nil {
			loop.epoll.regfd[fd] = regop
			return err
		}
		delete(loop.epoll.regfd, fd)
	} else if ok && regop == 0 {
		delete(loop.epoll.regfd, fd)
	}
	return nil
}

func (loop *AeEventLoop) AeCreateTimeEvent(milliseconds int, proc TimeProc, extra interface{}) int {
	// TODO
	return -123456789
}

func (loop *AeEventLoop) AeDeleteTimeEvent(id int) {
	//TODO
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

	// TODO timeEvent process
	{

	} //timeEvent process

	return processed
}

func (loop *AeEventLoop) aeWait() {
	//TODO : timeEvent wait
	n, err := unix.EpollWait(loop.epoll.epfd, loop.epoll.events, 10)
	if err != unix.EINTR {
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
