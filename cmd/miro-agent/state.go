package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/3nd3r1/miro/internal/status"
)

const (
	maxRolloutEvents = 20
	maxLogTailLines  = 50
)

// AgentState persists the offline site agent's reconciliation state across runs and reboots.
// Ordering relies solely on monotonic sequence numbers, never wall-clock time.
type AgentState struct {
	SiteID              string                `json:"site_id"`
	LastAppliedSequence uint64                `json:"last_applied_sequence"`
	CurrentSystem       string                `json:"current_system,omitempty"`
	CurrentGeneration   int                   `json:"current_generation"`
	CurrentModelSHA256  string                `json:"current_model_sha256,omitempty"`
	RolloutEvents       []status.RolloutEvent `json:"rollout_events,omitempty"`
	AgentLogTail        []string              `json:"agent_log_tail,omitempty"`
	ReportedRejections  map[string]string     `json:"reported_rejections,omitempty"`
}

// LoadAgentState loads persisted AgentState from path, or returns a new state if not found.
func LoadAgentState(path, siteID string) (*AgentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &AgentState{
				SiteID:             siteID,
				ReportedRejections: make(map[string]string),
			}, nil
		}
		return nil, fmt.Errorf("read agent state: %w", err)
	}

	var st AgentState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse agent state: %w", err)
	}
	if st.SiteID == "" {
		st.SiteID = siteID
	}
	if st.ReportedRejections == nil {
		st.ReportedRejections = make(map[string]string)
	}
	return &st, nil
}

// SaveAgentState writes AgentState atomically as JSON to path.
func SaveAgentState(path string, st *AgentState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal agent state: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

func (s *AgentState) appendEvent(ev status.RolloutEvent) {
	s.RolloutEvents = append(s.RolloutEvents, ev)
	if len(s.RolloutEvents) > maxRolloutEvents {
		s.RolloutEvents = s.RolloutEvents[len(s.RolloutEvents)-maxRolloutEvents:]
	}
}

func (s *AgentState) appendLog(line string) {
	s.AgentLogTail = append(s.AgentLogTail, line)
	if len(s.AgentLogTail) > maxLogTailLines {
		s.AgentLogTail = s.AgentLogTail[len(s.AgentLogTail)-maxLogTailLines:]
	}
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".agent-state-*")
	if err != nil {
		return fmt.Errorf("create temp state: %w", err)
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
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}
