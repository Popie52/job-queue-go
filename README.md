# job-queue-go

> A distributed, priority-based job queue built in Go — featuring PostgreSQL-backed persistence, crash-safe recovery, worker-pool concurrency, and transactional state management.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go&logoColor=white)
![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=flat&logo=postgresql&logoColor=white)
![Docker](https://img.shields.io/badge/Docker-ready-2496ED?style=flat&logo=docker&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-green?style=flat)

---

## Overview

job-queue-go implements a production-grade job queue with the reliability guarantees that real systems require:

- **No job loss** — all state is driven by PostgreSQL transactions, not in-memory assumptions
- **Priority scheduling with aging** — high-priority jobs run first; low-priority jobs never starve
- **Crash recovery** — on restart, pending and inflight jobs are automatically restored
- **Visibility timeout reprocessing** — stalled inflight jobs are detected and re-queued
- **Worker pool concurrency** — goroutine-based workers process jobs in parallel with bounded retries

---

## Load Test Results

Tested with concurrent job submission across multiple goroutine workers:

| Metric | Result |
|--------|--------|
| Total jobs processed | 1,000+ |
| Throughput | ~19 jobs/sec |
| Success rate | 93% |
| Recovery | Crashed inflight jobs auto-restored via visibility timeout |
| Retry handling | Bounded retries with exponential backoff |

> Run the load test yourself: `bash scripts/load_test.sh`

---

## Architecture

### High-Level Flow

![Architecture Diagram](diagram/d2.png)

### Components

| Component | Role |
|-----------|------|
| **Queue** | In-memory priority heap; used only for scheduling |
| **Dispatcher** | Coordinates between the in-memory queue and persistent store |
| **Store** | Pluggable backend — PostgreSQL implementation |
| **Workers** | Goroutine pool; each worker pulls, processes, and updates job state |
| **Metrics** | HTTP endpoint exposing runtime counters and gauges |

### Persistent State Model

Two PostgreSQL tables manage all job state:

```
pending_jobs   →   inflight_jobs   →   removed (done)
                       ↓
                 pending_jobs       (retry on failure)
```

All state transitions are performed inside PostgreSQL transactions — no partial updates, no lost jobs.

### Design Guarantees

- Job state is always database-authoritative; in-memory queue is secondary
- No job is lost on process crash or restart
- Inflight jobs past their visibility timeout are automatically recovered
- Retry paths are fully transactional with bounded attempt counts
- Graceful shutdown drains workers before exit

---

## Project Structure

```
job-queue-go/
├── cmd/
│   └── jobqueue/        # Main entrypoint
├── internal/
│   ├── queue/           # In-memory priority heap with aging
│   ├── dispatcher/      # Orchestrates queue ↔ store
│   ├── store/           # PostgreSQL persistence layer
│   ├── worker/          # Goroutine worker pool
│   └── metrics/         # HTTP metrics handler
├── scripts/
│   └── load_test.sh     # Load testing script
├── diagram/
│   └── d2.png           # Architecture diagram
├── schema.sql           # Database schema
├── Dockerfile
└── docker-compose.yml
```

---

## Quick Start

### Option 1 — Docker (Recommended)

No setup required. Spins up Go app + PostgreSQL 18 together:

```bash
docker-compose up --build
```

To stop:

```bash
docker-compose down
```

### Option 2 — Local Run

**Prerequisites:** Go 1.21+, PostgreSQL

```bash
# 1. Set database connection
export DATABASE_URL="postgres://USER:PASSWORD@localhost:5432/jobqueue?sslmode=disable"

# 2. Create tables
psql "$DATABASE_URL" -f schema.sql

# 3. Run server
go run ./cmd/jobqueue
```

---

## API Reference

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/submit` | Submit a new job |
| `GET` | `/metrics` | Queue health and throughput metrics |
| `GET` | `/health` | Liveness check |

### Submit a Job

```bash
curl -X POST http://localhost:8080/submit \
  -H "Content-Type: application/json" \
  -d '{"priority": 5, "max_retries": 3, "payload": {"task": "send_email"}}'
```

### Check Metrics

```bash
curl http://localhost:8080/metrics
```

---

## Configuration

Copy `.env.example` and adjust values:

```bash
cp .env.example .env
```

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | — | PostgreSQL connection string |
| `WORKER_COUNT` | `5` | Number of concurrent workers |
| `VISIBILITY_TIMEOUT` | `30s` | Time before inflight job is re-queued |
| `PORT` | `8080` | HTTP server port |

---

## Load Testing

```bash
bash scripts/load_test.sh
```

The script submits jobs concurrently and reports submission throughput, retry behavior, and completion rate.

---

## Database Schema

```sql
CREATE TABLE IF NOT EXISTS pending_jobs (
    id          TEXT PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL,
    priority    INT NOT NULL,
    payload     JSONB,
    attempts    INT NOT NULL,
    max_retries INT NOT NULL
);

CREATE TABLE IF NOT EXISTS inflight_jobs (
    id          TEXT PRIMARY KEY,
    created_at  TIMESTAMPTZ NOT NULL,
    priority    INT NOT NULL,
    payload     JSONB,
    attempts    INT NOT NULL,
    max_retries INT NOT NULL,
    picked_at   TIMESTAMPTZ NOT NULL
);
```

---

## Tech Stack

- **Language:** Go (goroutines, channels, sync primitives)
- **Database:** PostgreSQL 18 — transactional state, durable persistence
- **Infra:** Docker, docker-compose
- **Observability:** HTTP metrics endpoint

---

## License

MIT
