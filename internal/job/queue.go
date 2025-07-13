package job

import (
	"audioBot/internal/config"
	"context"
	"runtime"
)

// Processor defines the interface for processing a job request.
type Processor interface {
	Process(ctx context.Context, r Request) error
}

// Request holds the data for a job.
type Request struct {
	ChatID   int64
	FileID   string
	FileName string

	// For large files handled by TDLib
	IsLargeFile        bool
	ForwardedMessageID int
}

// Queue is a buffered channel that dispatches requests to workers.
type Queue struct {
	buf    chan Request
	ctx    context.Context
	cancel context.CancelFunc
}

// NewQueue creates a new queue and starts the worker pool.
func NewQueue(parent context.Context, cfg *config.Config, p Processor) *Queue {
	ctx, cancel := context.WithCancel(parent)
	q := &Queue{
		buf:    make(chan Request, cfg.QueueSize),
		ctx:    ctx,
		cancel: cancel,
	}

	workers := cfg.Workers
	if workers == 0 {
		workers = runtime.NumCPU() * 2
	}

	for i := 0; i < workers; i++ {
		go worker(ctx, q.buf, p)
	}

	return q
}

// Enqueue adds a request to the queue, blocking if the queue is full.
func (q *Queue) Enqueue(r Request) error {
	select {
	case q.buf <- r:
		return nil
	case <-q.ctx.Done():
		return q.ctx.Err()
	}
}

// Shutdown gracefully shuts down the queue and its workers.
func (q *Queue) Shutdown() {
	q.cancel()
}
