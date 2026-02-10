package store

import (
	"encoding/json"
	"github.com/Popie52/jobqueue/internal/model"
	"os"
	"sync"
)

type FileJobStore struct {
	pendingPath  string
	inflightPath string
	mu           sync.Mutex
}


func NewFileJobStore(pendingPath, inflightPath string) *FileJobStore {
	return &FileJobStore{
		pendingPath:  pendingPath,
		inflightPath: inflightPath,
	}
}

func (s *FileJobStore) SavePending(job *model.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	jobs, err := readJobs(s.pendingPath)
	if err != nil {
		return err
	}
	jobs = append(jobs, job)
	return writeJobs(s.pendingPath, jobs)
}

func (s *FileJobStore) LoadPending() ([]*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readJobs(s.pendingPath)
}

func (s *FileJobStore) LoadInFlight() ([]*model.Job, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return readJobs(s.inflightPath)
}

func (s *FileJobStore) MarkInFlight(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	pending, err := readJobs(s.pendingPath)
	if err != nil {
		return err
	}

	var (
		newPending  []*model.Job
		inflightAdd *model.Job
	)

	for _, j := range pending {
		if j.ID == jobID {
			inflightAdd = j
		} else {
			newPending = append(newPending, j)
		}
	}

	if inflightAdd == nil {
		return nil
	}
	inflight, err := readJobs(s.inflightPath)
	if err != nil {
		return err
	}
	inflight = append(inflight, inflightAdd)
	if err := writeJobs(s.pendingPath, newPending); err != nil {
		return err
	}
	return writeJobs(s.inflightPath, inflight)
}

func (s *FileJobStore) Remove(jobID string) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	pending, err := readJobs(s.pendingPath)
	if err != nil {
		return err
	}

	inflight, err := readJobs(s.inflightPath)
	if err != nil {
		return err
	}

	var newPending []*model.Job
	for _, j := range pending {
		if j.ID != jobID {
			newPending = append(newPending, j)
		}
	}

	var newInflight []*model.Job
	for _, j := range inflight {
		if j.ID != jobID {
			newInflight = append(newInflight, j)
		}
	}

	if err := writeJobs(s.pendingPath, newPending); err != nil {
		return err
	}

	return writeJobs(s.inflightPath, newInflight)
}

// Helpers
func readJobs(path string) ([]*model.Job, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*model.Job{}, nil
		}
		return nil, err
	}

	var jobs []*model.Job
	if err := json.Unmarshal(data, &jobs); err != nil {
		return nil, err
	}

	return jobs, nil
}

func writeJobs(path string, jobs []*model.Job) error {
	data, err := json.MarshalIndent(jobs, "", " ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
