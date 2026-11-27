package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// StatusFile is the filename of the status JSON payload inside a status bundle directory.
	StatusFile = "status.json"
)

// ErrInvalidStatus indicates a status bundle failed schema validation.
var ErrInvalidStatus = errors.New("invalid status bundle")

// Health reports the health of the local inference service on a site.
type Health struct {
	Healthy bool   `json:"healthy"`
	Message string `json:"message,omitempty"`
}

// RolloutEvent records one reconciliation attempt on a site.
type RolloutEvent struct {
	Sequence     uint64 `json:"sequence"`
	BundleID     string `json:"bundle_id,omitempty"`
	TargetSystem string `json:"target_system,omitempty"`
	ModelSHA256  string `json:"model_sha256,omitempty"`
	Outcome      string `json:"outcome"`
	Reason       string `json:"reason,omitempty"`
}

// Bundle represents the unsigned status report written by an offline site agent.
type Bundle struct {
	SiteID              string         `json:"site_id"`
	LastAppliedSequence uint64         `json:"last_applied_sequence"`
	CurrentSystem       string         `json:"current_system"`
	CurrentGeneration   int            `json:"current_generation"`
	CurrentModelSHA256  string         `json:"current_model_sha256,omitempty"`
	Health              Health         `json:"health"`
	RolloutEvents       []RolloutEvent `json:"rollout_events,omitempty"`
	AgentLogTail        []string       `json:"agent_log_tail,omitempty"`
}

// Validate checks required status bundle fields.
func (b *Bundle) Validate() error {
	if b == nil {
		return fmt.Errorf("%w: nil status bundle", ErrInvalidStatus)
	}
	if strings.TrimSpace(b.SiteID) == "" {
		return fmt.Errorf("%w: missing site_id", ErrInvalidStatus)
	}
	if b.CurrentGeneration < 0 {
		return fmt.Errorf("%w: negative current_generation %d", ErrInvalidStatus, b.CurrentGeneration)
	}
	for i, ev := range b.RolloutEvents {
		if strings.TrimSpace(ev.Outcome) == "" {
			return fmt.Errorf("%w: rollout_events[%d] missing outcome", ErrInvalidStatus, i)
		}
	}
	return nil
}

// WriteDir validates b and atomically writes status.json inside dir.
func WriteDir(dir string, b *Bundle) error {
	return WriteFile(filepath.Join(dir, StatusFile), b)
}

// ReadDir reads and validates status.json from a status bundle directory.
func ReadDir(dir string) (*Bundle, error) {
	return ReadFile(filepath.Join(dir, StatusFile))
}

// WriteFile validates b and atomically writes it as indented JSON to path.
func WriteFile(path string, b *Bundle) error {
	if err := b.Validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal status: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

// ReadFile reads, unmarshals, and validates a status bundle JSON file.
func ReadFile(path string) (*Bundle, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read status file: %w", err)
	}
	var b Bundle
	if err := json.Unmarshal(data, &b); err != nil {
		return nil, fmt.Errorf("%w: parse json: %w", ErrInvalidStatus, err)
	}
	if err := b.Validate(); err != nil {
		return nil, err
	}
	return &b, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".status-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp status file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp status file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp status file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp status file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp status file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp status file: %w", err)
	}
	cleanup = false
	return nil
}
