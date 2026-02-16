package db

import (
	"fmt"
	"log/slog"
	"sync"
)

// SagaStep represents a single forward action with a compensating rollback.
type SagaStep struct {
	Name       string       // human-readable label for logging
	Action     func() error // forward action (e.g. DB insert, FS write)
	Compensate func() error // rollback action (e.g. DB delete, FS remove)
}

// Saga orchestrates a sequence of steps where each step has a compensating
// action. If any step fails, all previously completed steps are rolled back
// in reverse order (the "saga pattern").
//
// Use this when you need to coordinate changes across different systems that
// don't share a single transaction boundary — for example, writing a DB record
// AND a file to disk.
type Saga struct {
	mu    sync.Mutex
	name  string
	steps []SagaStep
}

// NewSaga creates a new saga with the given name (used in log messages).
func NewSaga(name string) *Saga {
	return &Saga{name: name}
}

// AddStep appends a step to the saga.
func (s *Saga) AddStep(step SagaStep) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.steps = append(s.steps, step)
}

// Execute runs all steps in order. If a step fails, it compensates all
// previously completed steps in reverse order and returns the original error.
// Compensation errors are logged but do not replace the original error.
func (s *Saga) Execute() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var completed []SagaStep

	for i, step := range s.steps {
		slog.Debug("Saga executing step", "saga", s.name, "step", i+1, "total", len(s.steps), "name", step.Name)

		if err := step.Action(); err != nil {
			slog.Error("Saga step failed",
				"saga", s.name, "step", step.Name, "error", err, "completed", len(completed))

			// Compensate in reverse order
			for j := len(completed) - 1; j >= 0; j-- {
				cStep := completed[j]
				if cStep.Compensate == nil {
					continue
				}
				slog.Debug("Saga compensating step", "saga", s.name, "step", cStep.Name)
				if cErr := cStep.Compensate(); cErr != nil {
					slog.Error("Saga compensation failed", "saga", s.name, "step", cStep.Name, "error", cErr)
				}
			}

			return fmt.Errorf("saga %q failed at step %q: %w", s.name, step.Name, err)
		}

		completed = append(completed, step)
	}

	slog.Debug("Saga completed", "saga", s.name, "steps", len(s.steps))
	return nil
}
