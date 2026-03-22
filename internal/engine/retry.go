package engine

import (
	"math"
	"math/rand"
	"strconv"
	"time"

	"github.com/adrianguyareach/junglegreenattractor/internal/dot"
)

const (
	defaultInitialDelayMs = 200
	defaultBackoffFactor  = 2.0
	defaultMaxDelayMs     = 60000
)

type retryPolicy struct {
	maxAttempts    int
	initialDelayMs int
	backoffFactor  float64
	maxDelayMs     int
	jitter         bool
}

func buildRetryPolicy(node *dot.Node, graph *dot.Graph) retryPolicy {
	maxRetries := 0
	if v := node.Attr("max_retries", ""); v != "" {
		maxRetries, _ = strconv.Atoi(v)
	}
	if maxRetries == 0 {
		if v := graph.GraphAttr("default_max_retry", ""); v != "" {
			maxRetries, _ = strconv.Atoi(v)
		}
	}

	return retryPolicy{
		maxAttempts:    maxRetries + 1,
		initialDelayMs: defaultInitialDelayMs,
		backoffFactor:  defaultBackoffFactor,
		maxDelayMs:     defaultMaxDelayMs,
		jitter:         true,
	}
}

func (p retryPolicy) delayForAttempt(attempt int) time.Duration {
	delay := float64(p.initialDelayMs) * math.Pow(p.backoffFactor, float64(attempt-1))
	if delay > float64(p.maxDelayMs) {
		delay = float64(p.maxDelayMs)
	}
	if p.jitter {
		delay = delay * (0.5 + rand.Float64())
	}
	return time.Duration(delay) * time.Millisecond
}
