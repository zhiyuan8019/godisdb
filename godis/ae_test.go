package godis

import (
	"testing"
	"time"
)

func TestAeErr(t *testing.T) {
	aeError := AeErr{msg: "Test Error"}
	if aeError.Error() != "Test Error" {
		t.Error("AeErr.Error() method didn't return the correct error message.")
	}
	if !aeError.Is(aeError) {
		t.Error("AeErr.Is() method didn't recognize the same error.")
	}
	var target AeErr
	if !aeError.As(&target) {
		t.Error("AeErr.As() method didn't work correctly.")
	}
	if target.msg != "Test Error" {
		t.Error("AeErr.As() method didn't set the correct error message.")
	}
}

func TestAeCreateEventLoop(t *testing.T) {
	loop, err := AeCreateEventLoop()
	if err != nil {
		t.Fatalf("AeCreateEventLoop() returned an error: %v", err)
	}
	if loop == nil {
		t.Fatal("AeCreateEventLoop() returned a nil loop.")
	}
	if loop.epoll == nil {
		t.Fatal("AeCreateEventLoop() did not initialize the epoll state.")
	}
}

func TestAeStop(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	loop.AeStop()
	if !loop.stop {
		t.Error("AeStop() did not set the stop flag correctly.")
	}
}

func TestAeCreateFileEvent(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	err := loop.AeCreateFileEvent(1, AE_READABLE, nil, nil)
	if err != nil {
		t.Fatalf("AeCreateFileEvent() returned an error: %v", err)
	}
}

func TestAeDeleteFileEvent(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	loop.AeCreateFileEvent(1, AE_READABLE, nil, nil)
	err := loop.AeDeleteFileEvent(1, AE_READABLE, nil)
	if err != nil {
		t.Fatalf("AeDeleteFileEvent() returned an error: %v", err)
	}
}

func TestAeCreateTimeEvent(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	id := loop.AeCreateTimeEvent(1000, AE_NORMAL, nil, nil)
	if id <= 0 {
		t.Error("AeCreateTimeEvent() did not return a valid event ID.")
	}
}

func TestAeDeleteTimeEvent(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	id := loop.AeCreateTimeEvent(1000, AE_NORMAL, nil, nil)
	err := loop.AeDeleteTimeEvent(id)
	if err != nil {
		t.Fatalf("AeDeleteTimeEvent() returned an error: %v", err)
	}
}

func TestGetMsTime(t *testing.T) {
	msStart := GetMsTime()
	time.Sleep(10 * time.Millisecond)
	msEnd := GetMsTime()
	elapsed := msEnd - msStart
	if elapsed < 10 || elapsed > 15 {
		t.Error("GetMsTime() did not accurately measure time in milliseconds.")
	}
}

func TestAeEpollAdd(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	err := loop.aeEpollAdd(1, AE_READABLE)
	if err != nil {
		t.Fatalf("aeEpollAdd() returned an error: %v", err)
	}
}

func TestAeEpollDelete(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	loop.aeEpollAdd(1, AE_READABLE)
	err := loop.aeEpollDelete(1)
	if err != nil {
		t.Fatalf("aeEpollDelete() returned an error: %v", err)
	}
}

func TestAeProcessEvents(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	processed := loop.aeProcessEvents()
	if processed != 0 { // assuming no events are processed on a fresh loop
		t.Fatalf("aeProcessEvents() processed more events than expected: %v", processed)
	}
}

func TestAeSearchNearestTimer(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	loop.AeCreateTimeEvent(200, AE_NORMAL, nil, nil)
	loop.AeCreateTimeEvent(100, AE_NORMAL, nil, nil)
	nearest := loop.aeSearchNearestTimer()
	if nearest == nil || nearest.when != 100 {
		t.Error("aeSearchNearestTimer() did not find the nearest timer correctly.")
	}
}

func TestCalTimeInterval(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	loop.AeCreateTimeEvent(GetMsTime()+200, AE_NORMAL, nil, nil)
	interval := calTimeInterval(loop)
	if interval != 200 {
		t.Errorf("calTimeInterval() returned unexpected interval: %v", interval)
	}
}

func TestAeMain(t *testing.T) {
	loop, _ := AeCreateEventLoop()
	go loop.AeMain() // Start the event loop in a separate goroutine
	time.Sleep(10 * time.Millisecond)
	loop.AeStop() // Stop the event loop
}
