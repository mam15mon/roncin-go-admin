package server

import (
	"context"
	"time"
)

const backgroundWorkerMaxPollInterval = 15 * time.Second

type workerPollBackoff struct {
	initial time.Duration
	maximum time.Duration
	current time.Duration
}

func newWorkerPollBackoff(initial, maximum time.Duration) workerPollBackoff {
	return workerPollBackoff{initial: initial, maximum: maximum, current: initial}
}

func (b *workerPollBackoff) Next() time.Duration {
	interval := b.current
	b.current = min(b.current*2, b.maximum)
	return interval
}

func (b *workerPollBackoff) Reset() {
	b.current = b.initial
}

func waitForWorkerPoll(ctx context.Context, interval time.Duration) bool {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
