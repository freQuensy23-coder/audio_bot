package job

import (
	"context"
	"log"
)

// worker is a long-lived goroutine that processes jobs from the queue.
func worker(ctx context.Context, buf <-chan Request, p Processor) {
	for {
		select {
		case <-ctx.Done():
			return // Exit when the context is canceled.
		case r := <-buf:
			if err := p.Process(ctx, r); err != nil {
				log.Printf("Failed to process job for chat %d: %v", r.ChatID, err)
			}
		}
	}
}
