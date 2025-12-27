// Package examples provides HTTP handlers for example execution.
package examples

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ExampleMeta defines catalog metadata for an example.
type ExampleMeta struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Description      string `json:"description"`
	Group            string `json:"group"`
	EstimatedSeconds int    `json:"estimated_seconds"`
}

// API provides HTTP handlers for example execution.
type API struct {
	Jobs       *JobManager
	Examples   []*ExampleMeta
	examplesMu sync.RWMutex
	rng        *rand.Rand // Secure random for job IDs
}

// NewAPI creates a new examples API.
func NewAPI(jobs *JobManager) *API {
	if jobs == nil {
		jobs = NewJobManager(200) // Default capacity
	}
	// Initialize secure RNG for job ID generation
	var buf [8]byte
	var seed int64
	if _, err := cryptorand.Read(buf[:]); err != nil {
		// Fallback to time-based seed only if crypto/rand fails
		seed = time.Now().UnixNano()
	} else {
		// #nosec G115: int64 seed overflow from uint64 bytes is fine for RNG seeding
		seed = int64(binary.LittleEndian.Uint64(buf[:]))
	}
	api := &API{
		Jobs: jobs,
		// #nosec G404
		rng: rand.New(rand.NewSource(seed)),
	}
	api.seedExamples()
	return api
}

// RegisterRoutes registers endpoints on the router.
func (a *API) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/api/v1/beta/examples")
	g.GET("/catalog", a.Catalog)
	g.POST("/run", a.Run)
	g.GET("/run/:id", a.RunStatus)
	g.GET("/run/:id/logs", a.RunLogs)
	g.DELETE("/run/:id", a.RunCancel)
	g.GET("/jobs", a.RunJobs)
	g.POST("/composite/export/json", a.CompositeExportJSON)
	g.POST("/composite/export/csv", a.CompositeExportCSV)
}

func (a *API) seedExamples() {
	a.Examples = []*ExampleMeta{
		{ID: "gauth_protocol_basics:minimal_poa", Title: "Minimal PoA", Description: "Basic power-of-attorney construction", Group: "basics", EstimatedSeconds: 1},
		{ID: "gauth_protocol_basics:delegation", Title: "Delegation", Description: "Simple delegation chain", Group: "basics", EstimatedSeconds: 2},
		{ID: "gauth_protocol_basics:token", Title: "Token Issuance", Description: "Beta token creation", Group: "basics", EstimatedSeconds: 1},
		{ID: "advanced_poa:multi_level", Title: "Advanced Multi-level Delegation", Description: "Complex PoA scenario", Group: "advanced", EstimatedSeconds: 3},
		{ID: "negative:invalid_scope", Title: "Invalid Scope", Description: "Negative case: scope mismatch", Group: "negative", EstimatedSeconds: 1},
	}
}

// Catalog returns the list of available examples.
func (a *API) Catalog(c *gin.Context) {
	a.examplesMu.RLock()
	defer a.examplesMu.RUnlock()
	c.JSON(200, gin.H{"success": true, "examples": a.Examples, "count": len(a.Examples)})
}

// Run starts an example job (simulated) and returns job id.
func (a *API) Run(c *gin.Context) {
	var req struct {
		ID string `json:"id"`
	}
	if c.ShouldBindJSON(&req) != nil || req.ID == "" {
		c.JSON(400, gin.H{"success": false, "message": "missing id"})
		return
	}
	job := &ExampleJob{ID: a.randomNonce(8), ExampleID: req.ID, State: JobQueued, CreatedAt: time.Now()}
	a.Jobs.AddJob(job)
	a.Jobs.SetJobState(job.ID, JobRunning, "", "")
	a.Jobs.AppendLog(job.ID, "Starting example "+req.ID)
	// simulate asynchronous completion
	go func(id, ex string) {
		time.Sleep(500 * time.Millisecond)
		a.Jobs.AppendLog(id, "Executing...")
		time.Sleep(300 * time.Millisecond)
		a.Jobs.SetJobState(id, JobDone, "Example "+ex+" completed", "")
	}(job.ID, req.ID)
	c.JSON(202, gin.H{"success": true, "job_id": job.ID, "state": job.State})
}

// RunStatus returns current status of a job.
func (a *API) RunStatus(c *gin.Context) {
	id := c.Param("id")
	if j, ok := a.Jobs.GetJob(id); ok {
		c.JSON(200, gin.H{"success": true, "job": gin.H{"id": j.ID, "example_id": j.ExampleID, "state": j.State, "output": j.Output, "error": j.Error, "started_at": j.StartedAt, "finished_at": j.FinishedAt}})
		return
	}
	c.JSON(404, gin.H{"success": false, "message": "job not found"})
}

// RunLogs streams logs via SSE.
func (a *API) RunLogs(c *gin.Context) {
	id := c.Param("id")
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	// Allow clients to reconnect quickly if they wish
	c.Header("X-Accel-Buffering", "no") // for nginx proxies if present
	c.Writer.Flush()

	// Initial open signal & client reconnection backoff (3s) hint
	fmt.Fprint(c.Writer, ": open\n")      // comment style heartbeat marker
	fmt.Fprint(c.Writer, "retry: 3000\n") // advise EventSource to wait 3s before auto-reconnect
	fmt.Fprintf(c.Writer, "event: open\ndata: {\"ok\":true,\"job_id\":%q}\n\n", id)
	c.Writer.Flush()

	j, ok := a.Jobs.GetJob(id)
	if !ok {
		fmt.Fprintf(c.Writer, "event: done\ndata: {\"state\":\"not_found\"}\n\n")
		c.Writer.Flush()
		return
	}

	// Send initial status snapshot including any already captured logs
	statusPayload := map[string]any{"state": j.State, "output": j.Output, "error": j.Error, "job_id": j.ID}
	if b, err := json.Marshal(statusPayload); err == nil {
		fmt.Fprintf(c.Writer, "event: status\ndata: %s\n\n", b)
		c.Writer.Flush()
	}
	lastSent := 0
	if len(j.Logs) > 0 {
		for i := 0; i < len(j.Logs); i++ {
			fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", escapeSSEData(j.Logs[i]))
		}
		c.Writer.Flush()
		lastSent = len(j.Logs)
	}

	ticker := time.NewTicker(300 * time.Millisecond)
	heartbeat := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			// Minimal comment heartbeat to keep intermediaries from closing idle connection
			fmt.Fprint(c.Writer, ": ping\n\n")
			c.Writer.Flush()
		case <-ticker.C:
			j, ok := a.Jobs.GetJob(id)
			if !ok {
				fmt.Fprintf(c.Writer, "event: done\ndata: {\"state\":\"not_found\"}\n\n")
				c.Writer.Flush()
				return
			}
			logs := j.Logs
			for i := lastSent; i < len(logs); i++ {
				fmt.Fprintf(c.Writer, "event: log\ndata: %s\n\n", escapeSSEData(logs[i]))
			}
			if len(logs) > lastSent {
				c.Writer.Flush()
			}
			lastSent = len(logs)
			if j.State == JobDone || j.State == JobFailed || j.State == JobTimeout {
				donePayload := map[string]any{"state": j.State, "output": j.Output, "error": j.Error, "job_id": j.ID, "complete": true}
				if b, err := json.Marshal(donePayload); err == nil {
					fmt.Fprintf(c.Writer, "event: done\ndata: %s\n\n", b)
				} else {
					fmt.Fprintf(c.Writer, "event: done\ndata: {\"state\":\"%s\"}\n\n", j.State)
				}
				c.Writer.Flush()
				return
			}
		}
	}
}

// RunJobs returns a lightweight list of recent jobs for UI polling.
func (a *API) RunJobs(c *gin.Context) {
	// Optional limit query parameter (?limit=n)
	limitStr := c.Query("limit")
	var limit int
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
			limit = n
		}
	}
	jobs := a.Jobs.ListJobs(nil, limit) // nil state => all
	out := make([]gin.H, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, gin.H{
			"id":          j.ID,
			"example_id":  j.ExampleID,
			"state":       j.State,
			"started_at":  j.StartedAt,
			"finished_at": j.FinishedAt,
		})
	}
	c.JSON(200, gin.H{"success": true, "jobs": out, "count": len(out)})
}

// RunCancel attempts to cancel a running job (simulation: mark failed if running).
func (a *API) RunCancel(c *gin.Context) {
	id := c.Param("id")
	if j, ok := a.Jobs.GetJob(id); ok {
		if j.State == JobRunning || j.State == JobQueued {
			a.Jobs.SetJobState(id, JobFailed, "", "canceled")
		}
		c.JSON(200, gin.H{"success": true, "message": "cancel requested"})
		return
	}
	c.JSON(404, gin.H{"success": false, "message": "job not found"})
}

func (a *API) CompositeExportJSON(c *gin.Context) {
	var raw json.RawMessage
	if err := c.ShouldBindJSON(&raw); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid JSON payload"})
		return
	}
	c.Header("Content-Disposition", "attachment; filename=composite_run_summary.json")
	c.Data(http.StatusOK, "application/json", raw)
}

// CompositeExportCSV converts the composite run JSON array into a CSV file.
// Expected JSON schema: array of objects with fields: id, state, elapsed, output, error.
func (a *API) CompositeExportCSV(c *gin.Context) {
	var items []map[string]any
	if err := c.ShouldBindJSON(&items); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid JSON payload"})
		return
	}
	// Build CSV in-memory
	var b strings.Builder
	b.WriteString("id,state,elapsed,output,error\n")
	for _, it := range items {
		id := fmt.Sprint(it["id"])
		state := fmt.Sprint(it["state"])
		elapsed := fmt.Sprint(it["elapsed"])
		output := sanitizeCSV(fmt.Sprint(it["output"]))
		errVal := sanitizeCSV(fmt.Sprint(it["error"]))
		b.WriteString(id + "," + state + "," + elapsed + "," + output + "," + errVal + "\n")
	}
	c.Header("Content-Disposition", "attachment; filename=composite_run_summary.csv")
	c.Data(http.StatusOK, "text/csv; charset=utf-8", []byte(b.String()))
}

// sanitizeCSV escapes newlines and quotes minimally; not a full CSV implementation but
// adequate for safe returning of demo content.
func sanitizeCSV(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if strings.ContainsAny(s, ",\"") {
		return "\"" + strings.ReplaceAll(s, "\"", "'") + "\""
	}
	return s
}

func escapeSSEData(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "\r", " ")
}

// randomNonce generates a secure random nonce for job IDs.
// Uses crypto/rand-seeded RNG to ensure unpredictability.
func (a *API) randomNonce(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[a.rng.Intn(len(letters))]
	}
	return string(b)
}
