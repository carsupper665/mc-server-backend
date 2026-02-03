package service

import (
	"errors"
	"fmt"
	"go-backend/common"
	"go-backend/model"
	"strings"
	"sync"
	"time"
)

const (
	InstallSourceUser       = "user"
	InstallSourceDependency = "dependency"
)

type InstallJobStatus string

const (
	JobQueued    InstallJobStatus = "queued"
	JobRunning   InstallJobStatus = "running"
	JobCompleted InstallJobStatus = "completed"
	JobFailed    InstallJobStatus = "failed"
	JobCanceled  InstallJobStatus = "canceled"
)

type InstallRequest struct {
	ServerID   string
	WorkDir    string
	ModLoader  string
	GameVer    string
	ModID      string
	VersionID  string
	AutoUpdate bool
	Source     string
}

type InstallPlan struct {
	Items map[string]*PlanItem
	Order []InstallRequest
}

type PlanItem struct {
	Request InstallRequest
}

type InstallJob struct {
	mu        sync.Mutex
	ID        string
	ServerID  string
	Requests  []InstallRequest
	Plan      *InstallPlan
	Status    InstallJobStatus
	Progress  int
	Message   string
	ModID     string
	ModName   string
	Err       error
	CreatedAt time.Time
	StartedAt time.Time
	EndedAt   time.Time
	done      chan struct{}
}

func (j *InstallJob) setStatus(status InstallJobStatus, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	if err != nil {
		j.Err = err
	}
	switch status {
	case JobRunning:
		j.StartedAt = time.Now()
	case JobCompleted, JobFailed, JobCanceled:
		j.EndedAt = time.Now()
	}
}

func (j *InstallJob) wait() error {
	<-j.done
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.Err
}

type InstallJobSnapshot struct {
	ID        string           `json:"job_id"`
	ServerID  string           `json:"server_id"`
	Status    InstallJobStatus `json:"status"`
	Progress  int              `json:"progress"`
	Message   string           `json:"message,omitempty"`
	ModID     string           `json:"current_mod_id,omitempty"`
	ModName   string           `json:"current_mod_name,omitempty"`
	Error     string           `json:"error,omitempty"`
	CreatedAt time.Time        `json:"created_at"`
	StartedAt time.Time        `json:"started_at,omitempty"`
	EndedAt   time.Time        `json:"ended_at,omitempty"`
}

type serverQueue struct {
	mu      sync.Mutex
	running bool
	queue   []string
}

type installQueue struct {
	mu           sync.Mutex
	serverQueues map[string]*serverQueue
	jobs         map[string]*InstallJob
}

var modInstallQueue = &installQueue{
	serverQueues: map[string]*serverQueue{},
	jobs:         map[string]*InstallJob{},
}

var modNameCache = struct {
	mu   sync.Mutex
	data map[string]string
}{
	data: map[string]string{},
}

// EnqueueModInstall enqueues a mod install request (with dependency resolution).
// It returns a job that can be waited on for completion.
func EnqueueModInstall(serverID, workDir, modLoader, gameVer, modID, versionID string, autoUpdate bool) (*InstallJob, error) {
	if serverID == "" || modID == "" {
		return nil, errors.New("serverID and modID are required")
	}

	req := InstallRequest{
		ServerID:   serverID,
		WorkDir:    workDir,
		ModLoader:  modLoader,
		GameVer:    gameVer,
		ModID:      modID,
		VersionID:  versionID,
		AutoUpdate: autoUpdate,
		Source:     InstallSourceUser,
	}

	return modInstallQueue.enqueue(req), nil
}

// EnqueueModInstallAndWait enqueues a job and blocks until completion.
func EnqueueModInstallAndWait(serverID, workDir, modLoader, gameVer, modID, versionID string, autoUpdate bool) error {
	job, err := EnqueueModInstall(serverID, workDir, modLoader, gameVer, modID, versionID, autoUpdate)
	if err != nil {
		return err
	}
	return job.wait()
}

func (q *installQueue) enqueue(req InstallRequest) *InstallJob {
	sq := q.getServerQueue(req.ServerID)

	sq.mu.Lock()
	defer sq.mu.Unlock()

	// Try to merge into an existing queued job.
	for _, jobID := range sq.queue {
		q.mu.Lock()
		job := q.jobs[jobID]
		q.mu.Unlock()
		job.mu.Lock()
		status := job.Status
		job.mu.Unlock()
		if status == JobQueued {
			if err := mergeRequestIntoJob(job, req); err == nil {
				return job
			}
		}
	}

	job := &InstallJob{
		ID:        newInstallJobID(req.ServerID, req.ModID),
		ServerID:  req.ServerID,
		Requests:  []InstallRequest{req},
		Status:    JobQueued,
		CreatedAt: time.Now(),
		done:      make(chan struct{}),
	}
	ensureJobStream(job.ID)

	q.mu.Lock()
	q.jobs[job.ID] = job
	q.mu.Unlock()

	sq.queue = append(sq.queue, job.ID)
	if !sq.running {
		sq.running = true
		go q.worker(req.ServerID)
	}

	q.publishJobEvent(job, "queued", req, 0, "queued", nil)
	return job
}

func (q *installQueue) getServerQueue(serverID string) *serverQueue {
	q.mu.Lock()
	defer q.mu.Unlock()
	if sq, ok := q.serverQueues[serverID]; ok {
		return sq
	}
	sq := &serverQueue{}
	q.serverQueues[serverID] = sq
	return sq
}

func (q *installQueue) worker(serverID string) {
	sq := q.getServerQueue(serverID)
	for {
		sq.mu.Lock()
		if len(sq.queue) == 0 {
			sq.running = false
			sq.mu.Unlock()
			return
		}
		jobID := sq.queue[0]
		sq.queue = sq.queue[1:]
		sq.mu.Unlock()

		q.mu.Lock()
		job := q.jobs[jobID]
		q.mu.Unlock()

		job.setStatus(JobRunning, nil)
		q.publishJobEvent(job, "resolving", InstallRequest{ModID: ""}, 0, "resolving dependencies", nil)

		plan, err := buildInstallPlan(job.Requests)
		if err != nil {
			job.setStatus(JobFailed, err)
			q.publishJobEvent(job, "failed", InstallRequest{}, job.Progress, "resolve failed", err)
			CloseInstallEvents(job.ID)
			close(job.done)
			continue
		}
		job.mu.Lock()
		job.Plan = plan
		job.mu.Unlock()

		if err := executeInstallPlan(job, plan); err != nil {
			job.setStatus(JobFailed, err)
			q.publishJobEvent(job, "failed", InstallRequest{}, job.Progress, "install failed", err)
			CloseInstallEvents(job.ID)
			close(job.done)
			continue
		}

		job.setStatus(JobCompleted, nil)
		q.publishJobEvent(job, "completed", InstallRequest{}, 100, "completed", nil)
		CloseInstallEvents(job.ID)
		close(job.done)
	}
}

func newInstallJobID(serverID, modID string) string {
	return fmt.Sprintf("mod-install-%s-%s-%s", serverID, modID, common.GetTimeString())
}

func mergeRequestIntoJob(job *InstallJob, req InstallRequest) error {
	job.mu.Lock()
	defer job.mu.Unlock()

	for i := range job.Requests {
		if job.Requests[i].ModID == req.ModID {
			// conflict if different explicit versions
			if job.Requests[i].VersionID != "" && req.VersionID != "" && job.Requests[i].VersionID != req.VersionID {
				return fmt.Errorf("version conflict for %s: %s vs %s", req.ModID, job.Requests[i].VersionID, req.VersionID)
			}
			if job.Requests[i].VersionID == "" && req.VersionID != "" {
				job.Requests[i].VersionID = req.VersionID
			}
			if req.Source == InstallSourceUser {
				job.Requests[i].Source = InstallSourceUser
			}
			if req.AutoUpdate {
				job.Requests[i].AutoUpdate = true
			}
			return nil
		}
	}

	job.Requests = append(job.Requests, req)
	return nil
}

func (q *installQueue) publishJobEvent(job *InstallJob, stage string, req InstallRequest, percent int, message string, err error) {
	event := InstallEvent{
		JobID:     job.ID,
		ServerID:  job.ServerID,
		Stage:     stage,
		ModID:     req.ModID,
		ModName:   resolveModName(req.ModID),
		VersionID: req.VersionID,
		Percent:   percent,
		Message:   message,
	}
	if err != nil {
		event.Error = err.Error()
	}
	job.applyEvent(event)
	publishInstallEvent(event)
}

func (j *InstallJob) applyEvent(event InstallEvent) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if event.Percent >= 0 {
		j.Progress = event.Percent
	}
	if event.Message != "" {
		j.Message = event.Message
	}
	if event.ModID != "" {
		j.ModID = event.ModID
		j.ModName = event.ModName
	}
}

func (q *installQueue) hasJob(jobID string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.jobs[jobID]
	return ok
}

func (q *installQueue) getJob(jobID string) *InstallJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.jobs[jobID]
}

func GetInstallJobSnapshot(jobID string) (*InstallJobSnapshot, bool) {
	job := modInstallQueue.getJob(jobID)
	if job == nil {
		return nil, false
	}
	job.mu.Lock()
	defer job.mu.Unlock()
	snapshot := &InstallJobSnapshot{
		ID:        job.ID,
		ServerID:  job.ServerID,
		Status:    job.Status,
		Progress:  job.Progress,
		Message:   job.Message,
		ModID:     job.ModID,
		ModName:   job.ModName,
		CreatedAt: job.CreatedAt,
		StartedAt: job.StartedAt,
		EndedAt:   job.EndedAt,
	}
	if job.Err != nil {
		snapshot.Error = job.Err.Error()
	}
	return snapshot, true
}

func buildInstallPlan(reqs []InstallRequest) (*InstallPlan, error) {
	plan := &InstallPlan{
		Items: map[string]*PlanItem{},
		Order: []InstallRequest{},
	}

	visiting := map[string]bool{}
	cache := map[string]*ModrinthVersion{}

	for _, req := range reqs {
		if err := resolveInstallItem(plan, req, visiting, cache); err != nil {
			return nil, err
		}
	}
	return plan, nil
}

func resolveInstallItem(plan *InstallPlan, req InstallRequest, visiting map[string]bool, cache map[string]*ModrinthVersion) error {
	if req.ModID == "" {
		return errors.New("mod id is required")
	}

	if visiting[req.ModID] {
		return fmt.Errorf("dependency cycle detected for %s", req.ModID)
	}

	if item, ok := plan.Items[req.ModID]; ok {
		return mergePlanItem(item, req)
	}

	visiting[req.ModID] = true
	defer func() {
		visiting[req.ModID] = false
	}()

	version, err := resolveVersion(req, cache)
	if err != nil {
		return err
	}
	if !isCompatible(version, req.ModLoader, req.GameVer) {
		return UnSupportModErr
	}
	if req.VersionID == "" {
		req.VersionID = version.ID
	}

	for _, dep := range version.Dependencies {
		depType := strings.ToLower(strings.TrimSpace(dep.DependencyType))
		switch depType {
		case "required":
			depReq, err := dependencyToRequest(req, dep, cache)
			if err != nil {
				return err
			}
			if err := resolveInstallItem(plan, depReq, visiting, cache); err != nil {
				return err
			}
		case "incompatible":
			if plan.Items[dep.ProjectID] != nil {
				return fmt.Errorf("incompatible dependency detected: %s", dep.ProjectID)
			}
			if dep.ProjectID != "" {
				if ok, err := model.ModExists(req.ServerID, dep.ProjectID); err != nil {
					return err
				} else if ok {
					return fmt.Errorf("incompatible dependency installed: %s", dep.ProjectID)
				}
			}
		default:
			// optional/embedded/unknown: skip for now
			continue
		}
	}

	plan.Items[req.ModID] = &PlanItem{Request: req}
	plan.Order = append(plan.Order, req)
	return nil
}

func mergePlanItem(item *PlanItem, req InstallRequest) error {
	existing := item.Request
	if existing.VersionID != "" && req.VersionID != "" && existing.VersionID != req.VersionID {
		return fmt.Errorf("version conflict for %s: %s vs %s", req.ModID, existing.VersionID, req.VersionID)
	}
	if existing.VersionID == "" && req.VersionID != "" {
		existing.VersionID = req.VersionID
	}
	if req.Source == InstallSourceUser {
		existing.Source = InstallSourceUser
	}
	if req.AutoUpdate {
		existing.AutoUpdate = true
	}
	item.Request = existing
	return nil
}

func dependencyToRequest(parent InstallRequest, dep ModrinthDependency, cache map[string]*ModrinthVersion) (InstallRequest, error) {
	modID := dep.ProjectID
	versionID := dep.VersionID

	if modID == "" && versionID != "" {
		v, err := fetchModVersionByID(versionID)
		if err != nil {
			return InstallRequest{}, err
		}
		modID = v.ProjectID
		if versionID == "" {
			versionID = v.ID
		}
	}
	if modID == "" {
		return InstallRequest{}, errors.New("dependency project id missing")
	}

	return InstallRequest{
		ServerID:   parent.ServerID,
		WorkDir:    parent.WorkDir,
		ModLoader:  parent.ModLoader,
		GameVer:    parent.GameVer,
		ModID:      modID,
		VersionID:  versionID,
		AutoUpdate: parent.AutoUpdate,
		Source:     InstallSourceDependency,
	}, nil
}

func resolveVersion(req InstallRequest, cache map[string]*ModrinthVersion) (*ModrinthVersion, error) {
	if req.ModID == "" && req.VersionID == "" {
		return nil, errors.New("mod id or version id is required")
	}
	key := fmt.Sprintf("%s|%s|%s|%s", req.ModID, req.VersionID, req.ModLoader, req.GameVer)
	if v, ok := cache[key]; ok {
		return v, nil
	}

	var (
		version *ModrinthVersion
		err     error
	)

	if req.VersionID != "" {
		version, err = fetchModVersionByID(req.VersionID)
		if err != nil && req.ModID != "" {
			version, err = getLatestOrSpecific(req.ModID, req.ModLoader, req.GameVer, req.VersionID)
		}
	} else {
		version, err = getLatestOrSpecific(req.ModID, req.ModLoader, req.GameVer, "")
	}
	if err != nil {
		return nil, err
	}

	cache[key] = version
	return version, nil
}

func executeInstallPlan(job *InstallJob, plan *InstallPlan) error {
	total := len(plan.Order)
	if total == 0 {
		return nil
	}
	done := 0
	progress := func(done, total int) int {
		if total <= 0 {
			return 100
		}
		return int(float64(done) / float64(total) * 100)
	}

	for _, req := range plan.Order {
		pct := progress(done, total)
		modInstallQueue.publishJobEvent(job, "downloading", req, pct, "downloading", nil)
		err := AddMod(req.ServerID, req.WorkDir, req.ModLoader, req.GameVer, req.ModID, req.VersionID, req.AutoUpdate)
		if err != nil {
			if errors.Is(err, AlreadyInsErr) {
				done++
				pct = progress(done, total)
				modInstallQueue.publishJobEvent(job, "skipped", req, pct, "already installed", nil)
				continue
			}
			return err
		}
		done++
		pct = progress(done, total)
		modInstallQueue.publishJobEvent(job, "installed", req, pct, "installed", nil)
	}

	return nil
}

func resolveModName(modID string) string {
	if modID == "" {
		return ""
	}
	modNameCache.mu.Lock()
	if name, ok := modNameCache.data[modID]; ok {
		modNameCache.mu.Unlock()
		return name
	}
	modNameCache.mu.Unlock()

	if mod, err := model.GetModByID(modID); err == nil {
		if mod.Name != "" {
			modNameCache.mu.Lock()
			modNameCache.data[modID] = mod.Name
			modNameCache.mu.Unlock()
			return mod.Name
		}
	}

	if project, _, err := fetchModProject(modID); err == nil && project != nil {
		if project.Title != "" {
			modNameCache.mu.Lock()
			modNameCache.data[modID] = project.Title
			modNameCache.mu.Unlock()
			return project.Title
		}
	}

	return modID
}
