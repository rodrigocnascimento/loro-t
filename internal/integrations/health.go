package integrations

import "time"

// HealthStatus representa o estado observado para um componente integrado.
type HealthStatus string

const (
	HealthStatusOK       HealthStatus = "ok"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusDown     HealthStatus = "down"
	HealthStatusUnknown  HealthStatus = "unknown"
)

// HealthCheckResult é o resultado consolidado do monitoramento de saúde de um componente.
type HealthCheckResult struct {
	Component     string      `json:"component"`
	Status        HealthStatus `json:"status"`
	LastSuccessAt *time.Time  `json:"last_success_at,omitempty"`
	LastError     string      `json:"last_error,omitempty"`
	LatencyMS     int64       `json:"latency_ms"`
	TTLSeconds    int64       `json:"ttl_seconds"`
}

// Snapshot agrupa resultados por componente e registra quando foi produzido.
type Snapshot struct {
	GeneratedAt time.Time                     `json:"generated_at"`
	Components  map[string]HealthCheckResult `json:"components"`
}
