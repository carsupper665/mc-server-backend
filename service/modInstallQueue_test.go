package service

import "testing"

func TestCountPendingInstallJobs(t *testing.T) {
	jobs := []*InstallJob{
		newTestInstallJob("q", "srv-a", JobQueued),
		newTestInstallJob("r", "srv-a", JobRunning),
		newTestInstallJob("done", "srv-a", JobCompleted),
		newTestInstallJob("fail", "srv-a", JobFailed),
		newTestInstallJob("cancel", "srv-a", JobCanceled),
		newTestInstallJob("other", "srv-b", JobQueued),
	}
	modInstallQueue.mu.Lock()
	for _, job := range jobs {
		modInstallQueue.jobs[job.ID] = job
	}
	modInstallQueue.mu.Unlock()
	t.Cleanup(func() {
		modInstallQueue.mu.Lock()
		for _, job := range jobs {
			delete(modInstallQueue.jobs, job.ID)
		}
		modInstallQueue.mu.Unlock()
	})

	if got := CountPendingInstallJobs("srv-a"); got != 2 {
		t.Fatalf("CountPendingInstallJobs(srv-a)=%d, want 2", got)
	}
	if got := CountPendingInstallJobs("srv-b"); got != 1 {
		t.Fatalf("CountPendingInstallJobs(srv-b)=%d, want 1", got)
	}
	if got := CountPendingInstallJobs("none"); got != 0 {
		t.Fatalf("CountPendingInstallJobs(none)=%d, want 0", got)
	}
}

func newTestInstallJob(id, serverID string, status InstallJobStatus) *InstallJob {
	return &InstallJob{
		ID:       "count-test-" + id,
		ServerID: serverID,
		Status:   status,
		done:     make(chan struct{}),
	}
}
