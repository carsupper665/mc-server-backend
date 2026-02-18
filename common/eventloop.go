package common

import (
	"errors"
	"flag"
	"strings"
	"sync"
	"time"
)

var (
	ErrEventNameRequired = errors.New("event name is required")
	ErrTaskFuncNil       = errors.New("task func is nil")
	ErrIntervalInvalid   = errors.New("interval must be > 0")
	ErrCountInvalid      = errors.New("count must be >= 0 or -1")
)

type TaskFunc func() error

type event struct {
	f        TaskFunc
	name     string
	nextRun  time.Time
	Interval time.Duration
	count    int
}

var loopLogPath = flag.String("event-log-name", "event", "specify the log name")

type EventLoop struct {
	events   []event
	logger   *SysLogger
	sm       sync.Mutex
	stopBool bool
}

func NewEventLoop() (*EventLoop, error) {
	logger, err := NewSysLogger("EventThread", loopLogPath, 8000)
	if err != nil {
		return nil, err
	}
	return &EventLoop{
		events: make([]event, 0),
		logger: logger,
	}, nil
}

func (el *EventLoop) RegisterEvent(taskName string, f TaskFunc, interval time.Duration, count int) error {
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
	el.logger.Infof("Registering event: %s with interval: %s", taskName, interval)

	el.sm.Lock()
	defer el.sm.Unlock()

	el.events = append(el.events, event{
		f:        f,
		name:     taskName,
		nextRun:  time.Now().Add(interval),
		Interval: interval,
		count:    count,
	})
	return nil
}

func (el *EventLoop) runEvent(name string, f TaskFunc) {
	err := f()
	if err != nil {
		el.logger.Errorf("Error running event[%s]: %v", name, err)
	}
}

func (el *EventLoop) StopEventLoop() {
	el.sm.Lock()
	defer el.sm.Unlock()
	el.stopBool = true
}

func (el *EventLoop) cleanup() {

}

func (el *EventLoop) popExpireEvent(expIndex []int) {
	el.sm.Lock()
	defer el.sm.Unlock()
	if len(el.events) == 0 {
		return
	}
	if expIndex == nil || len(expIndex) == 0 {
		el.logger.Warn("EventLoop: pop expire event with empty expIndex")
		return
	}
	// 標記要刪除的 index，順便去重/過濾越界
	remove := make([]bool, len(el.events))
	validRemoveCount := 0
	for _, idx := range expIndex {
		if idx < 0 || idx >= len(el.events) || remove[idx] {
			continue
		}
		remove[idx] = true
		validRemoveCount++
	}

	if validRemoveCount == 0 {
		el.logger.Warn("EventLoop: no valid expired event index")
		return
	}

	newEvents := make([]event, 0, len(el.events)-validRemoveCount)

	for oldIdx, ev := range el.events {
		if remove[oldIdx] {
			el.logger.Infof("Removing expired event: %s", ev.name)
			continue
		}
		newEvents = append(newEvents, ev)
	}

	el.events = newEvents
}

func (el *EventLoop) Start() {
	el.logger.Infof("EventLoop starting...")
	minCheck := 1 * time.Second
	for { // while true
		el.sm.Lock()
		stop := el.stopBool
		el.sm.Unlock()
		if stop {
			el.logger.Info("EventLoop stopping...")
			el.cleanup()
			return
		}
		now := time.Now()
		checkEvery := minCheck
		due := make([]TaskFunc, 0, 8)
		dueName := make([]string, 0, 8)
		expEventIndex := make([]int, 0)

		el.sm.Lock()
		if len(el.events) > 0 {
			for i := range el.events {
				ev := &el.events[i]
				evName := ev.name

				if ev.count <= 0 && ev.count != -1 {
					expEventIndex = append(expEventIndex, i)
					continue
				}

				if !now.Before(ev.nextRun) {
					if ev.count != -1 {
						ev.count -= 1
					}
					// 到點就先更新 nextRun，再異步執行
					ev.nextRun = now.Add(ev.Interval)

					due = append(due, ev.f)
					dueName = append(dueName, evName)

				}
			}
		}
		el.sm.Unlock()

		for i, task := range due {
			go el.runEvent(dueName[i], task)
		}
		if len(expEventIndex) > 0 {
			el.popExpireEvent(expEventIndex)
		}

		el.sm.Lock()
		if len(el.events) == 0 {
			checkEvery = minCheck
		} else {
			checkEvery = el.events[0].Interval
			for i := 1; i < len(el.events); i++ {
				if el.events[i].Interval < checkEvery {
					checkEvery = el.events[i].Interval
				}
			}
			if checkEvery <= 0 {
				checkEvery = minCheck
			}
		}
		el.sm.Unlock()
		time.Sleep(checkEvery)
	}
}
