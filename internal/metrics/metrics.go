package metrics

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

type MetricsFn interface {
	IncJobsSubmitted()
	IncJobsCompleted()
	IncJobsFailed()
	IncJobsDead()
	IncJobsRetries()

	IncActiveWorkers()
	DecActiveWorkers()

	IncQueueDepth()
	DecQueueDepth()

	IncInflight()
	DecInflight()
}

type Metrics struct {
	// counters
	jobsSubmitted uint64
	jobsCompleted uint64
	jobsFailed    uint64
	jobsRetries   uint64
	jobsDead      uint64

	// gauges
	queueDepth int64
	inflight   int64
	activeW    int64
}

func New() *Metrics {
	return &Metrics{}
}

// counters
func (m *Metrics) IncJobsSubmitted() { atomic.AddUint64(&m.jobsSubmitted, 1) }
func (m *Metrics) IncJobsCompleted() { atomic.AddUint64(&m.jobsCompleted, 1) }
func (m *Metrics) IncJobsFailed()    { atomic.AddUint64(&m.jobsFailed, 1) }
func (m *Metrics) IncJobsDead()      { atomic.AddUint64(&m.jobsDead, 1) }
func (m *Metrics) IncJobsRetries()   { atomic.AddUint64(&m.jobsRetries, 1) }

// gauges
func (m *Metrics) IncQueueDepth() { atomic.AddInt64(&m.queueDepth, 1) }
func (m *Metrics) DecQueueDepth() { atomic.AddInt64(&m.queueDepth, -1) }

func (m *Metrics) IncInflight() { atomic.AddInt64(&m.inflight, 1) }
func (m *Metrics) DecInflight() { atomic.AddInt64(&m.inflight, -1) }

func (m *Metrics) IncActiveWorkers() { atomic.AddInt64(&m.activeW, 1) }
func (m *Metrics) DecActiveWorkers() { atomic.AddInt64(&m.activeW, -1) }

// Http handler

func (m *Metrics) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w,
			"jobs_submitted_total %d\n"+
				"jobs_completed_total %d\n"+
				"jobs_retries_total %d\n"+
				"jobs_failed_total %d\n"+
				"jobs_dead_total %d\n"+
				"queue_depth %d\n"+
				"inflight %d\n"+
				"active_workers %d\n", atomic.LoadUint64(&m.jobsSubmitted),
			atomic.LoadUint64(&m.jobsCompleted), atomic.LoadUint64(&m.jobsRetries),
			atomic.LoadUint64(&m.jobsFailed),
			atomic.LoadUint64(&m.jobsDead),
			atomic.LoadInt64(&m.queueDepth),
			atomic.LoadInt64(&m.inflight),
			atomic.LoadInt64(&m.activeW),
		)
	})
}
