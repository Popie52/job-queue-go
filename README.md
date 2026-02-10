# Job Queue (Go)

A production-style learning project implementing a high-priority job queue
with retries, persistence and graceful shutdown.

## Features

- In-memory priority queue with aging (starvation prevention)
- Worker pool with retry & backoff
- Postgres-backed transactional state
  - pending → inflight → done
- Crash recovery (pending + inflight restore on restart)
- Graceful shutdown (workers + HTTP server)
- Metrics exposed over HTTP

## Design Guarantees

- All job state transitions are driven by the database.
- In-memory queue is used only for scheduling.
- No job is lost on crash or restart.
- Retry and recovery paths are fully transactional.
- Inflight jobs are recovered automatically using a visibility timeout.

## Architecture

- Queue: in-memory priority heap
- Dispatcher: coordinates queue and persistent store
- Store: pluggable backend (Postgres implementation)
- Workers: concurrent job processors
- Metrics: runtime counters and gauges

## Persistent State Model

Two tables are used:

- `pending_jobs`
- `inflight_jobs`

State transition:

pending → inflight → removed

Retry transition:

inflight → pending

All state transitions are done using Postgres transactions.

## High level flow

                         ┌────────────────────┐
                         │     HTTP API       │
                         │   /submit, /health │
                         │     /metrics       │
                         └─────────┬──────────┘
                                   │
                                   │  Submit Job
                                   ▼
                         ┌────────────────────┐
                         │     Dispatcher     │
                         │  (queue + store)  │
                         └─────────┬──────────┘
                                   │
                                   │ SavePending()
                                   ▼
                      ┌─────────────────────────┐
                      │      PostgreSQL         │
                      │                         │
                      │   pending_jobs table    │
                      │   inflight_jobs table   │
                      └─────────┬──────────────┘
                                │
                                │ LoadPending() / recovery
                                ▼
                    ┌─────────────────────────────┐
                    │   In-memory Priority Queue  │
                    │     (heap + aging)          │
                    └─────────┬──────────────────┘
                              │
                              │ Pop()
                              ▼
                  ┌──────────────────────────────┐
                  │           Workers            │
                  │     (worker pool, goroutines)│
                  └─────────┬────────────────────┘
                            │
                            │ MarkInFlight()
                            ▼
                 ┌──────────────────────────────┐
                 │         PostgreSQL            │
                 │        inflight_jobs          │
                 └─────────┬────────────────────┘
                           │
        ┌──────────────────┼──────────────────────┐
        │                  │                      │
        ▼                  ▼                      ▼
   job completed       job failed            worker crash
        │                  │                      │
        │                  │                      │
        │                  ▼                      ▼
        │          Requeue (DB first)     Background recovery
        │           inflight → pending     (visibility timeout)
        │                  │                      │
        ▼                  ▼                      ▼
 Complete()           pending_jobs           pending_jobs
 (Remove from DB)         │
        │                 │
        └─────────────────┴───────────────►
                        back to
               In-memory Priority Queue

────────────────────────────────────────────────────────────

Background maintenance (separate goroutine):

   PostgreSQL inflight_jobs
          │
          │  picked_at < now() - timeout
          ▼
   RecoverStuckInFlight()
          │
          ▼
   move back to pending_jobs

## Requirements

- Go
- PostgreSQL

## Database Schema

See `schema.sql`.

## Run

Set database connection:

```bash
export DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/jobqueue?sslmode=disable"

Create tables:

psql "$DATABASE_URL" -f schema.sql


Run:

go run ./cmd/jobqueue

Endpoints

POST /submit – submit a job

GET /metrics – metrics

GET /health – health check


## Load testing

Basic load test scripts are available in the `scripts/` directory.
They were used to measure submission throughput and retry behavior.
