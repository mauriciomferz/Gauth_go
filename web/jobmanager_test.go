package web

import (
	"fmt"
	"testing"
	"time"
)

func TestJobManagerAddAndRetrieve(t *testing.T) {
	jm := NewJobManager(4)
	job := &ExampleJob{ID: "job1", ExampleID: "ex1", State: JobQueued, CreatedAt: time.Now()}
	jm.AddJob(job)
	got, ok := jm.GetJob("job1")
	if !ok {
		t.Fatalf("expected job to be retrievable")
	}
	if got.ID != job.ID || got.State != JobQueued {
		t.Fatalf("unexpected job fields: %+v", got)
	}
}

func TestJobManagerSetJobState(t *testing.T) {
	jm := NewJobManager(4)
	job := &ExampleJob{ID: "job2", ExampleID: "ex2", State: JobQueued, CreatedAt: time.Now()}
	jm.AddJob(job)
	jm.SetJobState("job2", JobRunning, "", "")
	if j, _ := jm.GetJob("job2"); j.State != JobRunning || j.StartedAt.IsZero() {
		t.Fatalf("expected job running with start time set: %+v", j)
	}
	jm.SetJobState("job2", JobDone, "output", "")
	j, _ := jm.GetJob("job2")
	if j.State != JobDone || j.Output != "output" || j.FinishedAt.IsZero() {
		t.Fatalf("expected job done with output and finish time: %+v", j)
	}
}

func TestJobManagerListJobs(t *testing.T) {
	jm := NewJobManager(10)
	// create jobs with staggered CreatedAt to test ordering
	for i := 0; i < 5; i++ {
		j := &ExampleJob{ID: fmt.Sprintf("j%d", i), ExampleID: "ex", State: JobQueued, CreatedAt: time.Now().Add(time.Duration(i) * time.Second)}
		jm.AddJob(j)
	}
	// move two jobs to running / done
	jm.SetJobState("j1", JobRunning, "", "")
	jm.SetJobState("j2", JobDone, "out", "")
	// list all
	all := jm.ListJobs(nil, 0)
	if len(all) != 5 {
		t.Fatalf("expected 5 jobs got %d", len(all))
	}
	// ensure ordering desc by CreatedAt
	for i := 1; i < len(all); i++ {
		if all[i].CreatedAt.After(all[i-1].CreatedAt) {
			t.Fatalf("list not sorted desc at index %d", i)
		}
	}
	// filter done
	f := JobDone
	done := jm.ListJobs(&f, 0)
	if len(done) != 1 || done[0].State != JobDone {
		t.Fatalf("expected 1 done job: %+v", done)
	}
	// limit
	limited := jm.ListJobs(nil, 2)
	if len(limited) != 2 {
		t.Fatalf("expected limit=2 got %d", len(limited))
	}
}
