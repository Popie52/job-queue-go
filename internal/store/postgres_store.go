package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Popie52/jobqueue/internal/model"
)

type PostgresJobStore struct {
	db *sql.DB
}

func NewPostgresJobStore(db *sql.DB) *PostgresJobStore {
	return &PostgresJobStore{
		db: db,
	}
}

func (s *PostgresJobStore) SavePending(job *model.Job) error {
	payload, err := json.Marshal(job.Payload)
	if err != nil {
		return err
	}

	_, err = s.db.Exec(`
		INSERT INTO pending_jobs (
			id,
			created_at,
			priority,
			payload,
			attempts,
			max_retries
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`,
		job.ID,
		job.CreatedAt,
		job.Priority,
		payload,
		job.Attempts,
		job.MaxRetries,
	)

	return err
}

func (s *PostgresJobStore) LoadPending() ([]*model.Job, error) {
	rows, err := s.db.Query(`
		SELECT 
			id,
			created_at,
			priority,
			payload,
			attempts,
			max_retries
		FROM pending_jobs
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*model.Job

	for rows.Next() {
		var (
			j       model.Job
			payload []byte
		)

		if err := rows.Scan(
			&j.ID,
			&j.CreatedAt,
			&j.Priority,
			&payload,
			&j.Attempts,
			&j.MaxRetries,
		); err != nil {
			return nil, err
		}

		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &j.Payload); err != nil {
				return nil, err
			}
		}

		jobs = append(jobs, &j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *PostgresJobStore) LoadInFlight() ([]*model.Job, error) {
	rows, err := s.db.Query(`
		SELECT 
			id,
			created_at,
			priority,
			payload,
			attempts,
			max_retries
		FROM inflight_jobs
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*model.Job

	for rows.Next() {
		var (
			j       model.Job
			payload []byte
		)

		if err := rows.Scan(
			&j.ID,
			&j.CreatedAt,
			&j.Priority,
			&payload,
			&j.Attempts,
			&j.MaxRetries,
		); err != nil {
			return nil, err
		}

		if len(payload) > 0 {
			if err := json.Unmarshal(payload, &j.Payload); err != nil {
				return nil, err
			}
		}

		jobs = append(jobs, &j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

func (s *PostgresJobStore) MarkInFlight(jobID string) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	var (
		id         string
		createdAt  time.Time
		priority   int
		payload    []byte
		attempts   int
		maxRetries int
	)

	row := tx.QueryRowContext(ctx, `
		SELECT 
			id,
			created_at,
			priority,
			payload,
			attempts,
			max_retries
		FROM pending_jobs
		WHERE id = $1
		FOR UPDATE
	`, jobID)

	err = row.Scan(
		&id,
		&createdAt,
		&priority,
		&payload,
		&attempts,
		&maxRetries,
	)

	if err == sql.ErrNoRows {
		_ = tx.Rollback()
		return nil
	}

	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM pending_jobs WHERE id = $1
	`, jobID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO inflight_jobs(id, created_at, priority, payload, attempts, max_retries, picked_at) 
		VALUES ($1, $2, $3, $4, $5, $6, now())
	`,
		id,
		createdAt,
		priority,
		payload,
		attempts,
		maxRetries,
	)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	err = tx.Commit()
	return err
}

func (s *PostgresJobStore) Remove(jobID string) error {

	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM pending_jobs
		WHERE id = $1
	`, jobID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	_, err = tx.ExecContext(ctx, `
		DELETE FROM inflight_jobs
		WHERE id = $1
	`, jobID)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (s *PostgresJobStore) RecoverStuckInFlight(cutoff time.Time) error {
	ctx := context.Background()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	rows, err := tx.QueryContext(ctx, `
	Select 
		id,
		created_at,
		priority,
		payload,
		attempts,
		max_retries
	FROM inflight_jobs
	where picked_at < $1
	For Update`, cutoff)
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	defer rows.Close()

	type rowData struct {
		id         string
		createdAt  time.Time
		priority   int
		payload    []byte
		attempts   int
		maxRetries int
	}

	var stuck []rowData
	for rows.Next() {
		var r rowData
		if err := rows.Scan(
			&r.id,
			&r.createdAt,
			&r.priority,
			&r.payload,
			&r.attempts,
			&r.maxRetries,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
		stuck = append(stuck, r)
	} 

	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return err 
	}


	for _, r := range stuck {
		if _, err := tx.ExecContext(ctx, `
		DELETE from inflight_jobs where id = $1`, r.id); err != nil {
			_ = tx.Rollback()
			return err
		}

		if _, err := tx.ExecContext(ctx, `
		INSERT into pending_jobs(id, created_at, priority, payload, attempts, max_retries) values($1,$2,$3,$4,$5,$6)`, 
			r.id,
			r.createdAt,
			r.priority,
			r.payload,
			r.attempts,
			r.maxRetries,
		); err != nil {
			_ = tx.Rollback()
			return err 
		}
	}

	return tx.Commit()
}
