package pipeline

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/chverma/siemagent/internal/classifier"
	"github.com/chverma/siemagent/internal/models"
)

// WorkerPool fans out LogEvents to N classifier goroutines and collects results.
type WorkerPool struct {
	n          int
	classifier classifier.Interface
	in         chan models.LogEvent
	out        chan models.ClassifiedEvent
	wg         sync.WaitGroup
}

func NewWorkerPool(n int, c classifier.Interface) *WorkerPool {
	return &WorkerPool{
		n:          n,
		classifier: c,
		in:         make(chan models.LogEvent, n*4),
		out:        make(chan models.ClassifiedEvent, n*4),
	}
}

// Start launches worker goroutines and returns the results channel.
// Call Close() after all events are submitted to signal end of input.
func (p *WorkerPool) Start(ctx context.Context) <-chan models.ClassifiedEvent {
	for i := range p.n {
		p.wg.Add(1)
		go func(id int) {
			defer p.wg.Done()
			for ev := range p.in {
				classified, err := p.classifier.Classify(ctx, ev)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[worker %d] classify error: %v\n", id, err)
					continue
				}
				p.out <- classified
			}
		}(i)
	}

	go func() {
		p.wg.Wait()
		close(p.out)
	}()

	return p.out
}

// Submit sends an event to the pool. Must be called after Start.
func (p *WorkerPool) Submit(ev models.LogEvent) {
	p.in <- ev
}

// Close signals no more events and waits for workers to drain.
func (p *WorkerPool) Close() {
	close(p.in)
}
