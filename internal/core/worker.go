package core

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/Popie52/jobqueue/internal/metrics"
	"github.com/Popie52/jobqueue/internal/model"
)

type Worker struct {
	id         int
	dispatcher *Dispatcher
	ctx        context.Context
	metrics    metrics.MetricsFn
}

func NewWorker(id int, dispatcher *Dispatcher, ctx context.Context, m metrics.MetricsFn) *Worker {
	return &Worker{
		id:         id,
		dispatcher: dispatcher,
		ctx:        ctx,
		metrics:    m,
	}
}

func (w *Worker) Run() {
	w.metrics.IncActiveWorkers()
	defer w.metrics.DecActiveWorkers()

	for {
		job, err := w.dispatcher.Pop(w.ctx)
		if err != nil {
			return
		}
		err = w.process(job)
		if err != nil {
			w.metrics.IncJobsFailed()
			w.handleFailure(job)
		} else {
			w.dispatcher.Complete(job.ID)
			w.metrics.IncJobsCompleted()
		}
	}
}

func (w *Worker) process(job *model.Job) error {
	fmt.Printf("worker %d processing job %ssubmit (priority=%d)\n", w.id, job.ID, job.Priority)

	if rand.Intn(2) == 0 {
		return fmt.Errorf("job failed")
	}

	println("job", job.ID, "completed")
	return nil
}

func (w *Worker) handleFailure(job *model.Job) {

	next := *job
	next.Attempts++

	if next.Attempts > next.MaxRetries {
		println("job", job.ID, "moved to dead (max retries exceeded)")
		w.dispatcher.Complete(job.ID)
		w.metrics.IncJobsDead()
		return
	}

	w.metrics.IncJobsRetries()

	delay := time.Duration(next.Attempts) * time.Second

	println("job", next.ID, "will retry after", delay.String())

	go func(j model.Job) {
		select {
		case <-time.After(delay):
			if err := w.dispatcher.Requeue(&j); err != nil {
				fmt.Println("requeue failed for job", j.ID, ":", err)
			}
		case <-w.ctx.Done():
			return
		}
	}(next)
}
