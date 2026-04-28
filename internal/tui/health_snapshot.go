package tui

import (
	"fmt"
	"sort"
	"time"

	"loro-t/internal/integrations"
)

// ComponentHealthView é a projeção pronta para renderização no TUI.
type ComponentHealthView struct {
	Component string
	Status    integrations.HealthStatus
	Age       string
	LatencyMS int64
	LastError string
}

// BuildHealthViews converte snapshot consolidado em linhas legíveis para a TUI.
func BuildHealthViews(snapshot integrations.Snapshot, now time.Time) []ComponentHealthView {
	keys := make([]string, 0, len(snapshot.Components))
	for k := range snapshot.Components {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	rows := make([]ComponentHealthView, 0, len(keys))
	for _, k := range keys {
		c := snapshot.Components[k]
		rows = append(rows, ComponentHealthView{
			Component: c.Component,
			Status:    c.Status,
			Age:       formatAge(c.LastSuccessAt, now),
			LatencyMS: c.LatencyMS,
			LastError: c.LastError,
		})
	}

	return rows
}

func formatAge(lastSuccessAt *time.Time, now time.Time) string {
	if lastSuccessAt == nil {
		return "n/a"
	}
	age := now.Sub(*lastSuccessAt)
	if age < 0 {
		age = 0
	}
	if age < time.Second {
		return "0s"
	}
	if age < time.Minute {
		return fmt.Sprintf("%ds", int(age.Seconds()))
	}
	if age < time.Hour {
		return fmt.Sprintf("%dm", int(age.Minutes()))
	}
	return fmt.Sprintf("%dh", int(age.Hours()))
}
