package provider

import "context"

type State string

const (
	StateDisabled State = "DISABLED"
	StateStopped  State = "STOPPED"
	StateStarting State = "STARTING"
	StateHealthy  State = "HEALTHY"
	StateDegraded State = "DEGRADED"
	StateDead     State = "DEAD"
)

type Model struct {
	ID       string
	Provider string
	Tools    bool
}

type Provider interface {
	ID() string
	State() State
	Health(ctx context.Context) error
	Models(ctx context.Context) ([]Model, error)
}