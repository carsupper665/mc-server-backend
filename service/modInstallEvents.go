package service

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const installEventBufferSize = 200

type InstallEvent struct {
	ID        int64     `json:"id"`
	JobID     string    `json:"job_id"`
	ServerID  string    `json:"server_id"`
	Stage     string    `json:"stage"`
	ModID     string    `json:"mod_id,omitempty"`
	ModName   string    `json:"mod_name,omitempty"`
	VersionID string    `json:"version_id,omitempty"`
	Percent   int       `json:"percent"`
	Message   string    `json:"message,omitempty"`
	Error     string    `json:"error,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type jobStream struct {
	mu          sync.Mutex
	subscribers map[chan InstallEvent]struct{}
	buffer      []InstallEvent
	closed      bool
}

type eventHub struct {
	mu      sync.Mutex
	streams map[string]*jobStream
}

var (
	installEventSeq  int64
	modInstallEvents = &eventHub{
		streams: map[string]*jobStream{},
	}
)

func ensureJobStream(jobID string) *jobStream {
	modInstallEvents.mu.Lock()
	defer modInstallEvents.mu.Unlock()
	if stream, ok := modInstallEvents.streams[jobID]; ok {
		return stream
	}
	stream := &jobStream{
		subscribers: map[chan InstallEvent]struct{}{},
		buffer:      []InstallEvent{},
	}
	modInstallEvents.streams[jobID] = stream
	return stream
}

func publishInstallEvent(ev InstallEvent) {
	if ev.JobID == "" {
		return
	}
	ev.ID = atomic.AddInt64(&installEventSeq, 1)
	if ev.Timestamp.IsZero() {
		ev.Timestamp = time.Now()
	}
	stream := ensureJobStream(ev.JobID)
	stream.mu.Lock()
	if stream.closed {
		stream.mu.Unlock()
		return
	}
	stream.buffer = append(stream.buffer, ev)
	if len(stream.buffer) > installEventBufferSize {
		stream.buffer = stream.buffer[len(stream.buffer)-installEventBufferSize:]
	}
	for ch := range stream.subscribers {
		select {
		case ch <- ev:
		default:
		}
	}
	stream.mu.Unlock()
}

func SubscribeInstallEvents(jobID string) (<-chan InstallEvent, []InstallEvent, func(), error) {
	if jobID == "" {
		return nil, nil, nil, errors.New("job id is required")
	}
	if !modInstallQueue.hasJob(jobID) {
		return nil, nil, nil, errors.New("job not found")
	}

	stream := ensureJobStream(jobID)
	ch := make(chan InstallEvent, 16)

	stream.mu.Lock()
	history := append([]InstallEvent(nil), stream.buffer...)
	if stream.closed {
		stream.mu.Unlock()
		close(ch)
		return ch, history, func() {}, nil
	}
	stream.subscribers[ch] = struct{}{}
	stream.mu.Unlock()

	unsubscribe := func() {
		stream.mu.Lock()
		if _, ok := stream.subscribers[ch]; ok {
			delete(stream.subscribers, ch)
			close(ch)
		}
		stream.mu.Unlock()
	}
	return ch, history, unsubscribe, nil
}

func CloseInstallEvents(jobID string) {
	modInstallEvents.mu.Lock()
	stream := modInstallEvents.streams[jobID]
	modInstallEvents.mu.Unlock()
	if stream == nil {
		return
	}
	stream.mu.Lock()
	if stream.closed {
		stream.mu.Unlock()
		return
	}
	stream.closed = true
	for ch := range stream.subscribers {
		close(ch)
		delete(stream.subscribers, ch)
	}
	stream.mu.Unlock()
}
