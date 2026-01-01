package service

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

type Snapshot struct {
	PID       int32     `json:"pid"`
	CPU       float64   `json:"cpu_percent"` // 可能 >100 (多核心)
	RSS       uint64    `json:"rss_bytes"`
	VMS       uint64    `json:"vms_bytes"`
	Threads   int32     `json:"threads"`
	Status    []string  `json:"status"`
	StartedAt time.Time `json:"started_at"`
	At        time.Time `json:"at"`
	Running   bool      `json:"running"`
	Err       string    `json:"err,omitempty"`
}

type Monitor struct {
	mu   sync.RWMutex
	last Snapshot

	p      *process.Process
	ctx    context.Context
	cancel context.CancelFunc
}

func SetUpMonitor(pid int32) (*Monitor, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Monitor{
		p:      p,
		ctx:    ctx,
		cancel: cancel,
		last: Snapshot{
			PID: pid,
			//Running: false,
		},
	}, nil
}

func (m *Monitor) Start(interval time.Duration) {
	// 暖機：Percent(0) 會用「上一次呼叫」做差分；第一次通常不準/可能為 0
	_, _ = m.p.Percent(0)

	t := time.NewTicker(interval)
	go func() {
		defer t.Stop()
		for {
			select {
			case <-m.ctx.Done():
				return
			case <-t.C:
				m.sampleOnce()
			}
		}
	}()
}

func (m *Monitor) Stop() { m.cancel() }

func (m *Monitor) Snapshot() Snapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.last
}
func (m *Monitor) sampleOnce() {
	now := time.Now()
	s := Snapshot{PID: m.last.PID, At: now}

	running, err := m.p.IsRunning()
	s.Running = running
	if err != nil {
		s.Err = err.Error()
	} else if !running {
		s.Err = "process not running"
	}

	// 就算 not running，仍可能拿到一些資訊；但通常可直接回錯
	if !running {
		m.setLast(s)
		return
	}

	if cpu, err := m.p.Percent(0); err == nil {
		s.CPU = cpu
	} else {
		s.Err = joinErr(s.Err, err)
	}

	if mi, err := m.p.MemoryInfo(); err == nil && mi != nil {
		s.RSS = mi.RSS
		s.VMS = mi.VMS
	} else if err != nil {
		s.Err = joinErr(s.Err, err)
	}

	if th, err := m.p.NumThreads(); err == nil {
		s.Threads = th
	} else if err != nil {
		s.Err = joinErr(s.Err, err)
	}

	if st, err := m.p.Status(); err == nil {
		s.Status = st
	} else if err != nil {
		s.Err = joinErr(s.Err, err)
	}

	if ct, err := m.p.CreateTime(); err == nil {
		// CreateTime 是毫秒 epoch
		s.StartedAt = time.UnixMilli(ct)
	} else if err != nil {
		s.Err = joinErr(s.Err, err)
	}

	m.setLast(s)
}

func (m *Monitor) setLast(s Snapshot) {
	m.mu.Lock()
	m.last = s
	m.mu.Unlock()
}

func joinErr(prev string, err error) string {
	if err == nil {
		return prev
	}
	if prev == "" {
		return err.Error()
	}
	return prev + "; " + err.Error()
}
