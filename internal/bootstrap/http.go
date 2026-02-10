package bootstrap

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Popie52/jobqueue/internal/metrics"
	"github.com/Popie52/jobqueue/internal/model"
	"github.com/Popie52/jobqueue/internal/queue"
	"github.com/Popie52/jobqueue/internal/store"
	"github.com/google/uuid"
)

type submitRequest struct {
	Priority   int `json:"priority"`
	MaxRetries int `json:"max_retries"`
}

func submitHandler(
	ctx context.Context,
	q *queue.Queue,
	st store.JobStore,
	m *metrics.Metrics,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		select {
		case <-ctx.Done():
			http.Error(w, "server is shutting down", http.StatusServiceUnavailable)
			return
		default:
		}

		if r.Method != http.MethodPost {
			http.Error(w, "only POST", http.StatusMethodNotAllowed)
			return
		}

		var req submitRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if req.Priority < 0 {
			http.Error(w, "priority must be >= 0", http.StatusBadRequest)
			return
		}

		maxRetries := req.MaxRetries
		if maxRetries == 0 {
			maxRetries = 3
		}

		job := &model.Job{
			ID: uuid.NewString(),
			Priority:   req.Priority,
			MaxRetries: maxRetries,
			CreatedAt:  time.Now(),
			Payload: req,
		}

		if err := st.SavePending(job); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		q.Push(job)

		m.IncJobsSubmitted()
		m.IncQueueDepth()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": job.ID})
	})
}
