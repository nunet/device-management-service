package eventhandler

import (
	"context"
	"log"
	"math"
	"sync"
	"time"
)

// default value for max number of retries
const defaultMaxRetries = 10

// Event represents the data needed to be sent to contract actor
type Event struct {
	ContractHostDID string
	ContractDID     string
	Payload         interface{}
	MaxRetries      int
	// private
	attempts int
}

// HandlerFunc defines the function signature for processing events.
type HandlerFunc func(event Event) error

// EventHandler manages event processing with concurrency and retries.
type EventHandler struct {
	events    chan Event
	wg        sync.WaitGroup
	workers   int
	handler   HandlerFunc
	baseDelay time.Duration
	maxDelay  time.Duration
	quit      chan struct{}
	ctx       context.Context
}

// New creates a new EventHandler.
// - workers: number of concurrent workers
// - queueSize: buffer size for the event channel
// - baseDelay: initial retry delay
// - maxDelay: maximum retry delay (cap for backoff)
// - handler: the function to process events
func New(ctx context.Context, workers, queueSize int, baseDelay, maxDelay time.Duration, handler HandlerFunc) *EventHandler {
	eh := &EventHandler{
		events:    make(chan Event, queueSize),
		workers:   workers,
		handler:   handler,
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
		quit:      make(chan struct{}),
		ctx:       ctx,
	}
	eh.start()
	return eh
}

// start launches worker goroutines.
func (eh *EventHandler) start() {
	for i := 0; i < eh.workers; i++ {
		eh.wg.Add(1)
		go eh.worker(i)
	}
}

// worker processes events from the channel.
func (eh *EventHandler) worker(id int) {
	defer eh.wg.Done()
	for {
		select {
		case event := <-eh.events:
			err := eh.handler(event)
			if err != nil {
				log.Printf("[Worker %d] Error: %v (attempt %d/%d)", id, err, event.attempts+1, event.MaxRetries)
				if event.attempts < event.MaxRetries {
					event.attempts++
					delay := eh.backoff(event.attempts)
					go func(e Event, d time.Duration) {
						select {
						case <-time.After(d):
							eh.Push(e)
						case <-eh.ctx.Done():
							return
						}
					}(event, delay)
				}
			}
		case <-eh.quit:
			log.Printf("[Worker %d] Stopping (quit signal)", id)
			return
		case <-eh.ctx.Done():
			log.Printf("[Worker %d] Stopping (context done)", id)
			return
		}
	}
}

// Push adds an event to the queue.
func (eh *EventHandler) Push(event Event) {
	if event.MaxRetries == 0 {
		event.MaxRetries = defaultMaxRetries
	}

	go func() {
		// may block if queue is full, but only this goroutine blocks
		// this is fine as long as we dont have millions of events in the queue
		eh.events <- event
	}()
}

// Stop gracefully shuts down the event handler.
func (eh *EventHandler) Stop() {
	close(eh.quit)
	eh.wg.Wait()
}

// backoff calculates exponential backoff delay.
func (eh *EventHandler) backoff(attempt int) time.Duration {
	delay := float64(eh.baseDelay) * math.Pow(2, float64(attempt-1))
	if delay > float64(eh.maxDelay) {
		return eh.maxDelay
	}
	return time.Duration(delay)
}
