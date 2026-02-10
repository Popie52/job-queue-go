# Job Queue (Go)

A production-style learning project implementing a high-priority job queue
with retries, persistence, and graceful shutdown.

## Features

- In-memory priority queue with aging (starvation prevention)
- Worker pool with retry & backoff
- Postgres-backed transactional state
  - pending → inflight → done
- Crash recovery (pending + inflight restore on restart)
- Graceful shutdown (workers + HTTP server)
- Metrics exposed over HTTP

## Architecture & Design

### Design Guarantees

- All job state transitions are driven by the database.
- In-memory queue is used only for scheduling.
- No job is lost on crash or restart.
- Retry and recovery paths are fully transactional.
- Inflight jobs are recovered automatically using a visibility timeout.

### Components

- Queue: in-memory priority heap
- Dispatcher: coordinates queue and persistent store
- Store: pluggable backend (Postgres implementation)
- Workers: concurrent job processors
- Metrics: runtime counters and gauges

### Persistent State Model

Two tables:

- `pending_jobs`
- `inflight_jobs`

State transitions:

pending → inflight → removed
inflight → pending (retry)

All transitions are done using Postgres transactions.

### High-level Flow

![Diagram](diagram/d2.png)

## Requirements

- Go
- PostgreSQL

## Database Schema

See `schema.sql`.

## Run

```bash
# Set database connection
export DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/jobqueue?sslmode=disable"

# Create tables
psql "$DATABASE_URL" -f schema.sql

# Run server
go run ./cmd/jobqueue

| Method | Path     | Description  |
| ------ | -------- | ------------ |
| POST   | /submit  | Submit a job |
| GET    | /metrics | Metrics      |
| GET    | /health  | Health check |


Load Testing
Basic load test scripts are available in the scripts/ directory; they measure
submission throughput and retry behavior.


---
