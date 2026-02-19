package common

import (
	"errors"
	"flag"
	"fmt"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

var (
	ErrEventNameRequired = errors.New("event name is required")
	ErrTaskFuncNil       = errors.New("task func is nil")
	ErrIntervalInvalid   = errors.New("interval must be > 0")
	ErrCountInvalid      = errors.New("count must be >= 0 or -1")
	ErrEventNotFound     = errors.New("event not found")
	ErrEventNotAsync     = errors.New("you cant wait for a non-async event")
	ErrAlreadyExist      = errors.New("event with the same name already exists")
)

type TaskFunc func() error

type event struct {
	f        TaskFunc
	name     string
	nextRun  time.Time
	Interval time.Duration
	count    int
	done     chan error
}

var loopLogPath = flag.String("event-log-name", "event", "specify the log name")

type EventLoop struct {
	events   map[string]event
	logger   *SysLogger
	sm       sync.RWMutex
	stopBool bool
	running  bool
}

func NewEventLoop() (*EventLoop, error) {
	logger, err := NewSysLogger("EventThread", loopLogPath, 8000)
	if err != nil {
		return nil, err
	}
	return &EventLoop{
		events: make(map[string]event),
		logger: logger,
	}, nil
}

func (el *EventLoop) RegisterEvent(taskName string, f TaskFunc, interval time.Duration, count int) error {
	if err := el.validateRegisterReq(taskName, f, interval, count); err != nil {
		return err
	}
	el.logger.Infof("Registering event: %s with interval: %s", taskName, interval)

	el.sm.Lock()
	defer el.sm.Unlock()
	// 不用 IsEventExists 因為已經拿到鎖了，直接檢查 map 就好
	_, exists := el.events[taskName]
	if exists {
		return ErrAlreadyExist
	}

	el.appendEvent(taskName, f, interval, count, false)
	return nil
}

func (el *EventLoop) validateRegisterReq(taskName string, f TaskFunc, interval time.Duration, count int) error {
	if strings.TrimSpace(taskName) == "" {
		return ErrEventNameRequired
	}
	if f == nil {
		return ErrTaskFuncNil
	}
	if interval <= 0 {
		return ErrIntervalInvalid
	}
	if count <= 0 && count != -1 {
		return ErrCountInvalid
	}
	if interval < 1*time.Second && count == -1 {
		el.logger.Warnf(
			"Periodic task %q interval=%s (<1s). If runtime exceeds the interval, tasks can pile up and cause high CPU usage. Increase the interval or set a max-run count.",
			taskName, interval,
		)
	}
	return nil
}

func (el *EventLoop) isEventExists(taskName string) bool {
	el.sm.RLock()
	defer el.sm.RUnlock()
	_, exists := el.events[taskName]
	return exists
}

func (el *EventLoop) IsEmpty() bool {
	el.sm.RLock()
	defer el.sm.RUnlock()
	return len(el.events) == 0
}

func (el *EventLoop) appendEvent(taskName string, f TaskFunc, interval time.Duration, count int, async bool) {
	var done chan error
	if async {
		done = make(chan error, 1)
	}
	el.events[taskName] = event{
		f:        f,
		name:     taskName,
		nextRun:  time.Now().Add(interval),
		Interval: interval,
		count:    count,
		done:     done,
	}
}

func (el *EventLoop) RegisterAsyncEvent(taskName string, f TaskFunc, interval time.Duration, count int) error {
	if err := el.validateRegisterReq(taskName, f, interval, count); err != nil {
		return err
	}
	el.logger.Infof("Registering async event: %s with interval: %s", taskName, interval)
	el.sm.Lock()
	defer el.sm.Unlock()
	_, exists := el.events[taskName]
	if exists {
		return ErrAlreadyExist
	}
	el.appendEvent(taskName, f, interval, count, true)
	return nil
}

func sendAsyncResult(done chan error, err error) {
	if done == nil {
		return
	}
	select {
	case done <- err:
	default:
		// Keep only the latest result to avoid blocking periodic tasks.
		select {
		case <-done:
		default:
		}
		select {
		case done <- err:
		default:
		}
	}
}

func (el *EventLoop) runEvent(name string, f TaskFunc, done chan error) {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in event[%s]: %v", name, r)
			el.logger.Errorf("Panic running event[%s]: %v\n%s", name, r, debug.Stack())
		}
		sendAsyncResult(done, err)
	}()
	err = f()
	if err != nil {
		el.logger.Errorf("Error running event[%s]: %v", name, err)
	}
	sendAsyncResult(done, err)
}

func (el *EventLoop) Wait(name string) error {
	el.sm.Lock()
	e, ok := el.events[name]
	el.sm.Unlock()

	if !ok {
		el.logger.Warnf("EventLoop: no event found with name: %s", name)
		return ErrEventNotFound
	}
	if e.done == nil {
		return ErrEventNotAsync
	}
	err := <-e.done
	return err
}

func (el *EventLoop) StopEventLoop() {
	el.sm.Lock()
	defer el.sm.Unlock()
	el.stopBool = true
	el.running = false
	el.logger.Info("Stopping EventLoop...")
}

func (el *EventLoop) cleanup() {
	// for future use, e.g. close log files, release resources, etc.
	el.logger.Info("Cleaning up EventLoop resources...")
}

func (el *EventLoop) popExpireEvent(expNames []string) {
	el.sm.Lock()
	defer el.sm.Unlock()
	if len(el.events) == 0 || len(expNames) == 0 {
		return
	}
	for _, name := range expNames {
		ev, ok := el.events[name]
		if !ok {
			continue
		}
		el.logger.Infof("Removing expired event: %s", ev.name)
		delete(el.events, name)
	}
}

func (el *EventLoop) DelEvent(name string) {
	if e := el.IsEmpty(); e {
		el.logger.Warn("Event loop is empty")
		return
	}
	if !el.isEventExists(name) {
		el.logger.Warnf("EventLoop: no event found with name: %s", name)
		return
	}

	el.logger.Infof("Deleting event: %s", name)
	el.sm.Lock()
	defer el.sm.Unlock()
	delete(el.events, name)
}

func (el *EventLoop) mainLoop(minCheck time.Duration) {
	// main loop
	for { // while true
		el.sm.Lock()
		stop := el.stopBool
		el.sm.Unlock()
		if stop {
			el.sm.Lock()
			el.running = false
			el.sm.Unlock()
			el.logger.Info("EventLoop stopping...")
			el.cleanup()
			return
		}

		now := time.Now()
		checkEvery := minCheck
		due := make([]event, 0, 8)
		expEventName := make([]string, 0)

		el.sm.Lock()
		if len(el.events) > 0 {
			for name, ev := range el.events {
				if ev.count <= 0 && ev.count != -1 {
					expEventName = append(expEventName, name)
					continue
				}
				if !now.Before(ev.nextRun) {
					if ev.count != -1 {
						ev.count -= 1
					}
					// 到點就先更新 nextRun，再異步執行
					ev.nextRun = now.Add(ev.Interval)
					el.events[name] = ev
					due = append(due, ev)
				}
			}
		}
		el.sm.Unlock()

		for _, ev := range due {
			go el.runEvent(ev.name, ev.f, ev.done)
		}

		if len(expEventName) > 0 {
			el.popExpireEvent(expEventName)
		}

		time.Sleep(checkEvery)
	}

}

// Start starts the event loop in a separate goroutine. It will check for due events at least every minCheck(1s) duration.
func (el *EventLoop) Start() {
	el.sm.Lock()
	if el.running {
		el.sm.Unlock()
		el.logger.Warn("EventLoop is already running")
		return
	}
	el.stopBool = false
	el.running = true
	el.sm.Unlock()

	el.logger.Infof("EventLoop starting...")
	minCheck := 50 * time.Millisecond
	go el.mainLoop(minCheck)
	return
}
