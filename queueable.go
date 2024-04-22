package queue

import (
	"context"
	"time"
)

type Queueable interface {
	ID() string
	Handle(ctx context.Context) error
	ShouldRetry() bool
	RetryCount() int
	Retry()
	Timeout() time.Duration
}
