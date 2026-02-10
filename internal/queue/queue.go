package queue

import (
	"container/heap"
	"context"
	"sync"
	"time"
	"github.com/Popie52/jobqueue/internal/model"
)

// index to track
type jobItem struct {
	job   *model.Job
	index int
}

// actual container
type priorityQueue []*jobItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return effectivePriority(pq[i].job) > effectivePriority(pq[j].job) } // effective priority

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(newJob any) {
	item := newJob.(*jobItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)

	item := old[n-1]
	old[n-1] = nil

	*pq = old[:n-1]
	return item
}


type Queue struct {
	mu sync.Mutex
	cond *sync.Cond
	pq priorityQueue
}


// Constructor
func NewQueue() *Queue {
	q := &Queue{}
	q.cond = sync.NewCond(&q.mu)
	heap.Init(&q.pq)
	return q
}

func (q *Queue) Push(job *model.Job) {
	q.mu.Lock()
	defer q.mu.Unlock()

	heap.Push(&q.pq, &jobItem{job: job})
	q.cond.Signal()
}


func (q *Queue) Pop(ctx context.Context) (*model.Job, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	for q.pq.Len() == 0 {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		q.cond.Wait()
	}
	heap.Init(&q.pq) //ignore current order and recheck for order again here 
	// Takes O(n) time

	item := heap.Pop(&q.pq).(*jobItem)
	return item.job, nil
}

func effectivePriority(j *model.Job) int {
	wait := int(time.Since(j.CreatedAt).Seconds())
	return j.Priority + wait
}

func (q *Queue) Shutdown() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.cond.Broadcast()
}
