package plagiarism

import (
	"context"
	"runtime"
	"sync"

	"github.com/rs/zerolog/log"
)

type Job interface {
	Execute(ctx context.Context) error
}

type WorkerPool struct {
	workers  int
	jobQueue chan Job
	wg       sync.WaitGroup
	ctx      context.Context
	cancel   context.CancelFunc
}

// creates a new worker pool with CPU-based sizing
func NewWorkerPool(ctx context.Context) *WorkerPool {
	totalCPU := runtime.NumCPU()
	systemReserve := max(1, totalCPU/4) // Reserve 1/4 of the CPU for system processes
	size := totalCPU - systemReserve
	log.Info().
		Int("totalCPU", totalCPU).
		Int("systemReserve", systemReserve).
		Int("workers", size).
		Msg("Worker pool initialized")
	poolCtx, cancel := context.WithCancel(ctx)

	pool := &WorkerPool{
		workers:  size,
		jobQueue: make(chan Job, size*2), // Buffer 2x the worker count
		ctx:      poolCtx,
		cancel:   cancel,
	}

	// Start workers
	pool.start()

	return pool
}

// starts all worker goroutines
func (p *WorkerPool) start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// worker goroutine that processes jobs
func (p *WorkerPool) worker(id int) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				return // Channel closed
			}
			if err := job.Execute(p.ctx); err != nil {
				log.Error().Err(err).Msg("Worker failed to execute job")
			}
		}
	}
}

// submits a job to the pool
func (p *WorkerPool) Submit(job Job) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()
	case p.jobQueue <- job:
		return nil
	}
}

// closes the worker pool and waits for all workers to finish
func (p *WorkerPool) Close() {
	close(p.jobQueue)
	p.cancel()
	p.wg.Wait()
}

// returns the number of workers
func (p *WorkerPool) Size() int {
	return p.workers
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
