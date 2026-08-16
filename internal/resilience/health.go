package resilience

import (
	"context"
	"net/http"
	"time"
)

type HealthProber struct {
	targetURL string
	interval  time.Duration
	client    *http.Client
	cb        *CircuitBreaker
}

func NewHealthProber(
	targetURL string,
	interval time.Duration,
	client *http.Client,
	cb *CircuitBreaker,
) *HealthProber {
	if client == nil {
		client = &http.Client{
			Timeout: 2 * time.Second,
		}
	}

	if interval <= 0 {
		interval = 10 * time.Second
	}

	return &HealthProber{
		targetURL: targetURL,
		interval:  interval,
		client:    client,
		cb:        cb,
	}
}

func (hp *HealthProber) Start(ctx context.Context) {
	if hp == nil || hp.cb == nil {
		return
	}

	ticker := time.NewTicker(hp.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hp.check(ctx)

		case <-ctx.Done():
			return
		}
	}
}

func (hp *HealthProber) check(ctx context.Context) {
	if hp == nil || hp.cb == nil {
		return
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		hp.targetURL,
		nil,
	)
	if err != nil {
		hp.cb.RecordFailure()
		return
	}

	resp, err := hp.client.Do(req)
	if err != nil {
		hp.cb.RecordFailure()
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK ||
		resp.StatusCode >= http.StatusMultipleChoices {
		hp.cb.RecordFailure()
		return
	}

	hp.cb.RecordSuccess()
}
