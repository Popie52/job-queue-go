package store

import (
	"github.com/Popie52/jobqueue/internal/model"
)

type JobStore interface {
	SavePending(job *model.Job) error

	MarkInFlight(jobID string) error

	Remove(jobID string) error

	LoadPending() ([]*model.Job, error)

	LoadInFlight() ([]*model.Job, error)
}