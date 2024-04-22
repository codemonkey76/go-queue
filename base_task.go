package queue

import "time"

type BaseTask struct {
	TaskID           string
	_retryCount      int
	MaxRetries       int
	ExecutionTimeout time.Duration
}

func (bt *BaseTask) ID() string {
	return bt.TaskID
}

func (bt *BaseTask) ShouldRetry() bool {
	return bt._retryCount < bt.MaxRetries
}

func (bt *BaseTask) Retry() {
	bt._retryCount++
}

func (bt *BaseTask) RetryCount() int {
	return bt._retryCount
}

func (bt *BaseTask) Timeout() time.Duration {
	return bt.ExecutionTimeout
}
