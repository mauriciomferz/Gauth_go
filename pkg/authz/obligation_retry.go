package authz

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// RetryConfig defines the configuration for the retry mechanism.
type RetryConfig struct {
	MaxRetries       int
	BaseDelay        time.Duration
	MaxDelay         time.Duration
	AsyncQueueSize   int
	AsyncWorkerCount int
}

// DefaultRetryConfig provides sensible defaults.
var DefaultRetryConfig = RetryConfig{
	MaxRetries:       3,
	BaseDelay:        100 * time.Millisecond,
	MaxDelay:         5 * time.Second,
	AsyncQueueSize:   1000,
	AsyncWorkerCount: 5,
}

// RetryingObligationExecutor wraps an ObligationExecutor with retry logic.
type RetryingObligationExecutor struct {
	inner  ObligationExecutor
	config RetryConfig

	asyncQueue chan asyncRetryTask
	wg         sync.WaitGroup
	quit       chan struct{}
}

type asyncRetryTask struct {
	obligation Obligation
	context    map[string]interface{}
	attempt    int
}

// NewRetryingObligationExecutor creates a new executor with the given config.
func NewRetryingObligationExecutor(inner ObligationExecutor, config RetryConfig) *RetryingObligationExecutor {
	if config.MaxRetries < 0 {
		config.MaxRetries = 0
	}
	if config.AsyncQueueSize <= 0 {
		config.AsyncQueueSize = 100
	}
	if config.AsyncWorkerCount <= 0 {
		config.AsyncWorkerCount = 1
	}

	exec := &RetryingObligationExecutor{
		inner:      inner,
		config:     config,
		asyncQueue: make(chan asyncRetryTask, config.AsyncQueueSize),
		quit:       make(chan struct{}),
	}

	exec.startWorkers()
	return exec
}

// StartWorkers spins up the background workers.
func (e *RetryingObligationExecutor) startWorkers() {
	for i := 0; i < e.config.AsyncWorkerCount; i++ {
		e.wg.Add(1)
		go e.worker()
	}
}

// Stop cleanly drains the queue and stops workers.
func (e *RetryingObligationExecutor) Stop() {
	close(e.quit)
	e.wg.Wait()
}

// Execute attempts to execute the obligation with retries.
func (e *RetryingObligationExecutor) Execute(ob Obligation, ctx map[string]interface{}) error {
	if ob.Mandatory {
		return e.executeSync(ob, ctx)
	}
	// For non-mandatory, try once synchronously. If fail, queue for async retry.
	err := e.inner.Execute(ob, ctx)
	if err == nil {
		return nil
	}

	// Initial execution failed, queue for async retry
	select {
	case e.asyncQueue <- asyncRetryTask{obligation: ob, context: ctx, attempt: 1}:
		// Queued successfully, return nil (since it's not mandatory)
		log.Printf("[OBLIGATION] Queued for retry: %s (error: %v)", ob.ID, err)
		return nil
	default:
		// Queue full, log and drop (non-mandatory)
		log.Printf("[OBLIGATION] Async retry queue full, dropping obligation: %s", ob.ID)
		e.inner.PersistAudit(ob, ctx, fmt.Errorf("dropped due to full retry queue: %v", err))
		return nil
	}
}

// PersistAudit delegates to the inner executor.
func (e *RetryingObligationExecutor) PersistAudit(ob Obligation, ctx map[string]interface{}, result error) error {
	return e.inner.PersistAudit(ob, ctx, result)
}

// executeSync handles mandatory obligations with synchronous retry.
func (e *RetryingObligationExecutor) executeSync(ob Obligation, ctx map[string]interface{}) error {
	var err error
	for attempt := 0; attempt <= e.config.MaxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(e.backoffDuration(attempt))
		}
		err = e.inner.Execute(ob, ctx)
		if err == nil {
			return nil
		}
		log.Printf("[OBLIGATION] Sync retry attempt %d/%d failed: %v", attempt, e.config.MaxRetries, err)
	}
	return err // Return last error
}

// worker processes the async queue.
func (e *RetryingObligationExecutor) worker() {
	defer e.wg.Done()
	for {
		select {
		case <-e.quit:
			return
		case task := <-e.asyncQueue:
			e.processTask(task)
		}
	}
}

func (e *RetryingObligationExecutor) processTask(task asyncRetryTask) {
	// Execute
	err := e.inner.Execute(task.obligation, task.context)
	if err == nil {
		return
	}

	// Failed
	if task.attempt >= e.config.MaxRetries {
		log.Printf("[OBLIGATION] Max async retries reached for %s: %v", task.obligation.ID, err)
		// Final audit log for failure
		e.inner.PersistAudit(task.obligation, task.context, err)
		return
	}

	// Backoff and Re-queue
	go func() {
		time.Sleep(e.backoffDuration(task.attempt))
		newTask := task
		newTask.attempt++
		select {
		case e.asyncQueue <- newTask:
		case <-e.quit:
			// System stopping, drop
		}
	}()
}

func (e *RetryingObligationExecutor) backoffDuration(attempt int) time.Duration {
	// Exponential backoff: Base * 2^(attempt-1)
	factor := math.Pow(2, float64(attempt-1))
	delay := time.Duration(float64(e.config.BaseDelay) * factor)
	if delay > e.config.MaxDelay {
		delay = e.config.MaxDelay
	}
	return delay
}
