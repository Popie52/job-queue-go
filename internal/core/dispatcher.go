package core

import (
	"context"

	"github.com/Popie52/jobqueue/internal/metrics"
	"github.com/Popie52/jobqueue/internal/model"
	"github.com/Popie52/jobqueue/internal/queue"
	"github.com/Popie52/jobqueue/internal/store"
)

type Dispatcher struct {
	queue *queue.Queue
	store store.JobStore
	metrics metrics.MetricsFn
}

func NewDispatcher(q *queue.Queue, store store.JobStore, m metrics.MetricsFn) *Dispatcher {
	return &Dispatcher{
		queue: q,
		store: store,
		metrics: m,
	}
}

func (d *Dispatcher) Pop(ctx context.Context) (*model.Job, error) {
	job, err := d.queue.Pop(ctx)
	if err != nil {
		return nil, err
	}

	d.metrics.DecQueueDepth()

	if err := d.store.MarkInFlight(job.ID); err != nil {
		d.queue.Push(job)
		d.metrics.IncQueueDepth()
		return nil, err
	}

	d.metrics.IncInflight()
	return job, nil
}

func (d *Dispatcher) Requeue(job *model.Job) error {
	if err := d.store.Remove(job.ID); err != nil {
		return err 
	}

	if err := d.store.SavePending(job); err != nil {
		return err 
	}

	d.metrics.DecInflight()
	d.queue.Push(job)
	d.metrics.IncQueueDepth()
	
	return nil 
}

func (d *Dispatcher) Complete(jobID string) {
	d.store.Remove(jobID)
	d.metrics.DecInflight()
}
