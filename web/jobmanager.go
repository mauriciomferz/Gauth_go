package web

import (
	"sort"
	"sync"
	"time"
)

// JobState represents state of an example job.
type JobState string

const (
	JobQueued  JobState = "queued"
	JobRunning JobState = "running"
	JobDone    JobState = "done"
	JobFailed  JobState = "failed"
	JobTimeout JobState = "timeout"
)

// ExampleJob represents execution metadata.
type ExampleJob struct {
	ID         string
	ExampleID  string
	State      JobState
	Output     string
	Error      string
	CreatedAt  time.Time
	StartedAt  time.Time
	FinishedAt time.Time
	Logs       []string // incremental log lines for SSE
}

// JobManager stores jobs in memory.
type JobManager struct {
	mu       sync.RWMutex
	jobs     map[string]*ExampleJob
	order    []*ExampleJob // for ordering by CreatedAt desc
	capacity int
}

// NewJobManager creates manager with capacity (0 => unlimited).
func NewJobManager(capacity int) *JobManager {
	return &JobManager{jobs: map[string]*ExampleJob{}, capacity: capacity}
}

func (jm *JobManager) AddJob(j *ExampleJob) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	jm.jobs[j.ID] = j
	jm.order = append(jm.order, j)
	jm.resort()
	jm.enforceCapacity()
}

func (jm *JobManager) GetJob(id string) (*ExampleJob, bool) {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	j, ok := jm.jobs[id]
	return j, ok
}

func (jm *JobManager) SetJobState(id string, st JobState, out, errStr string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if j, ok := jm.jobs[id]; ok {
		if st == JobRunning && j.StartedAt.IsZero() {
			j.StartedAt = time.Now()
		}
		if st == JobDone || st == JobFailed || st == JobTimeout {
			j.FinishedAt = time.Now()
		}
		j.State = st
		if out != "" {
			j.Output = out
		}
		if errStr != "" {
			j.Error = errStr
		}
	}
}

// AppendLog appends a log line to a job if it exists.
func (jm *JobManager) AppendLog(id, line string) {
	jm.mu.Lock()
	defer jm.mu.Unlock()
	if j, ok := jm.jobs[id]; ok {
		j.Logs = append(j.Logs, line)
	}
}

// GetLogs returns a copy of logs for a job.
func (jm *JobManager) GetLogs(id string) []string {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	if j, ok := jm.jobs[id]; ok {
		out := make([]string, len(j.Logs))
		copy(out, j.Logs)
		return out
	}
	return nil
}

// ListJobs returns jobs filtered by optional state and limited.
func (jm *JobManager) ListJobs(state *JobState, limit int) []*ExampleJob {
	jm.mu.RLock()
	defer jm.mu.RUnlock()
	var out []*ExampleJob
	for _, j := range jm.order {
		if state != nil && j.State != *state {
			continue
		}
		out = append(out, j)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func (jm *JobManager) resort() {
	sort.Slice(jm.order, func(i, j int) bool {
		return jm.order[i].CreatedAt.After(jm.order[j].CreatedAt)
	})
}

func (jm *JobManager) enforceCapacity() {
	if jm.capacity <= 0 || len(jm.order) <= jm.capacity {
		return
	}
	// order is desc so scan end for overflow
	for len(jm.order) > jm.capacity {
		last := jm.order[len(jm.order)-1]
		delete(jm.jobs, last.ID)
		jm.order = jm.order[:len(jm.order)-1]
	}
}
