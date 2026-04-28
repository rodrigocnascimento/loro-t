package daemon

import (
	"context"
	"sort"
	"sync"
	"time"

	"loro-t/internal/integrations"
)

// CheckFunc executa um check e retorna latência medida e eventual erro.
type CheckFunc func(ctx context.Context) (time.Duration, error)

type componentMonitor struct {
	name          string
	ttl           time.Duration
	baseInterval  time.Duration
	maxBackoff    time.Duration
	consecFailure int
	nextRun       time.Time
	check         CheckFunc
	result        integrations.HealthCheckResult
}

// HealthScheduler executa checks periódicos com backoff simples e mantém snapshot consolidado.
type HealthScheduler struct {
	mu         sync.RWMutex
	components map[string]*componentMonitor
	now        func() time.Time
}

func NewHealthScheduler() *HealthScheduler {
	return &HealthScheduler{
		components: make(map[string]*componentMonitor),
		now:        time.Now,
	}
}

func (s *HealthScheduler) RegisterComponent(name string, ttl, interval, maxBackoff time.Duration, check CheckFunc) {
	if maxBackoff < interval {
		maxBackoff = interval
	}
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()
	s.components[name] = &componentMonitor{
		name:         name,
		ttl:          ttl,
		baseInterval: interval,
		maxBackoff:   maxBackoff,
		nextRun:      now,
		check:        check,
		result: integrations.HealthCheckResult{
			Component:  name,
			Status:     integrations.HealthStatusUnknown,
			TTLSeconds: int64(ttl.Seconds()),
		},
	}
}

func (s *HealthScheduler) Run(ctx context.Context) {
	t := time.NewTicker(200 * time.Millisecond)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *HealthScheduler) tick(ctx context.Context) {
	now := s.now()

	s.mu.RLock()
	checks := make([]*componentMonitor, 0, len(s.components))
	for _, c := range s.components {
		if !now.Before(c.nextRun) {
			checks = append(checks, c)
		}
	}
	s.mu.RUnlock()

	for _, c := range checks {
		s.runComponentCheck(ctx, c.name)
	}
}

func (s *HealthScheduler) runComponentCheck(ctx context.Context, name string) {
	s.mu.RLock()
	c, ok := s.components[name]
	s.mu.RUnlock()
	if !ok {
		return
	}

	latency, err := c.check(ctx)
	now := s.now()

	s.mu.Lock()
	defer s.mu.Unlock()
	c = s.components[name]
	if c == nil {
		return
	}

	c.result.LatencyMS = latency.Milliseconds()
	if err == nil {
		c.consecFailure = 0
		c.result.Status = integrations.HealthStatusOK
		c.result.LastError = ""
		lastSuccess := now
		c.result.LastSuccessAt = &lastSuccess
		c.nextRun = now.Add(c.baseInterval)
		return
	}

	c.consecFailure++
	c.result.LastError = err.Error()
	c.result.Status = transitionFailureStatus(c.result.Status, c.consecFailure)

	backoff := c.baseInterval << min(c.consecFailure, 3)
	if backoff > c.maxBackoff {
		backoff = c.maxBackoff
	}
	c.nextRun = now.Add(backoff)

	if c.result.LastSuccessAt != nil && now.Sub(*c.result.LastSuccessAt) > (2*c.ttl) {
		c.result.Status = integrations.HealthStatusDown
	}
}

func (s *HealthScheduler) Snapshot() integrations.Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	components := make(map[string]integrations.HealthCheckResult, len(s.components))
	for name, c := range s.components {
		components[name] = c.result
	}
	return integrations.Snapshot{
		GeneratedAt: s.now(),
		Components:  components,
	}
}

func (s *HealthScheduler) SnapshotList() []integrations.HealthCheckResult {
	snap := s.Snapshot()
	out := make([]integrations.HealthCheckResult, 0, len(snap.Components))
	for _, v := range snap.Components {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Component < out[j].Component })
	return out
}

func transitionFailureStatus(current integrations.HealthStatus, failures int) integrations.HealthStatus {
	if failures >= 3 {
		return integrations.HealthStatusDown
	}
	switch current {
	case integrations.HealthStatusUnknown:
		return integrations.HealthStatusDegraded
	case integrations.HealthStatusOK:
		return integrations.HealthStatusDegraded
	case integrations.HealthStatusDegraded:
		return integrations.HealthStatusDown
	default:
		return integrations.HealthStatusDown
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
