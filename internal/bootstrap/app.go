package bootstrap

import (
	"context"
	"database/sql"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Popie52/jobqueue/internal/core"
	"github.com/Popie52/jobqueue/internal/queue"
	"github.com/Popie52/jobqueue/internal/store"

	"net/http"

	"github.com/Popie52/jobqueue/internal/metrics"
	_ "github.com/lib/pq"
)

func Run() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sig
		log.Println("shutdown signal received")
		cancel()
	}()

	// metrics

	m := metrics.New()

	// store

	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	defer db.Close()

	store := store.NewPostgresJobStore(db)

	go func() {

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cutoff := time.Now().Add(-30 * time.Second)
				if err := store.RecoverStuckInFlight(cutoff); err != nil {
					log.Println("recover stuck inflight failed:", err)
				}
			}
		}
	}()

	// queue
	q := queue.NewQueue()

	// load from disk (restart recovery)
	restoreJobs(q, store)

	// dispatcher
	dispatcher := core.NewDispatcher(q, store, m)

	// http Handlers
	mux := http.NewServeMux()
	mux.Handle("/metrics", m.Handler())
	mux.Handle("/submit", submitHandler(ctx, q, store, m))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("http server error:", err)
		}
	}()

	// gracefull http shutdown
	go func() {
		<-ctx.Done()

		log.Println("shutting down http server...")
		q.Shutdown()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_ = srv.Shutdown(shutdownCtx)
	}()

	w1 := core.NewWorker(1, dispatcher, ctx, m)
	w2 := core.NewWorker(2, dispatcher, ctx, m)

	var wg sync.WaitGroup

	wg.Add(2)

	go func() {
		defer wg.Done()
		w1.Run()
	}()

	go func() {
		defer wg.Done()
		w2.Run()
	}()

	log.Println("job queue started")
	<-ctx.Done()

	log.Println("waiting for workers to stop...")
	wg.Wait()

	log.Println("bootstrap exiting")
}

func restoreJobs(q *queue.Queue, store store.JobStore) {
	pending, err := store.LoadPending()
	if err != nil {
		panic(err)
	}

	inflight, err := store.LoadInFlight()
	if err != nil {
		panic(err)
	}

	for _, j := range pending {
		q.Push(j)
	}
	for _, j := range inflight {
		q.Push(j)
	}
}
