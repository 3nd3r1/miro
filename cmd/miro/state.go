package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/3nd3r1/miro/internal/status"
)

const defaultStatePath = ".miro/state.json"

// HubState persists desired and actual state for all sites managed by the hub.
type HubState struct {
	Sites map[string]*SiteState `json:"sites"`
}

// SiteState tracks outbound exports and imported status reports for one site.
type SiteState struct {
	SiteID               string           `json:"site_id"`
	LastExportedSequence uint64           `json:"last_exported_sequence"`
	Desired              *DesiredState    `json:"desired,omitempty"`
	Actual               *ActualState     `json:"actual,omitempty"`
	PendingBundles       []ExportedBundle `json:"pending_bundles,omitempty"`
}

// DesiredState records the latest exported target for a site.
type DesiredState struct {
	Sequence    uint64 `json:"sequence"`
	BundleID    string `json:"bundle_id"`
	SystemPath  string `json:"system_path"`
	ModelSHA256 string `json:"model_sha256,omitempty"`
	ExportedAt  string `json:"exported_at,omitempty"`
}

// ExportedBundle tracks an outbound bundle waiting for a status confirmation.
type ExportedBundle struct {
	BundleID    string `json:"bundle_id"`
	Sequence    uint64 `json:"sequence"`
	SystemPath  string `json:"system_path"`
	ModelSHA256 string `json:"model_sha256,omitempty"`
	OutputDir   string `json:"output_dir,omitempty"`
	ExportedAt  string `json:"exported_at,omitempty"`
}

// ActualState mirrors the latest imported status bundle from a site.
type ActualState struct {
	LastAppliedSequence uint64                `json:"last_applied_sequence"`
	CurrentSystem       string                `json:"current_system"`
	CurrentGeneration   int                   `json:"current_generation"`
	CurrentModelSHA256  string                `json:"current_model_sha256,omitempty"`
	Health              status.Health         `json:"health"`
	RolloutEvents       []status.RolloutEvent `json:"rollout_events,omitempty"`
	AgentLogTail        []string              `json:"agent_log_tail,omitempty"`
	ImportedAt          string                `json:"imported_at,omitempty"`
}

// LoadState reads HubState from path, returning an empty state if the file does not exist.
func LoadState(path string) (*HubState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &HubState{Sites: make(map[string]*SiteState)}, nil
		}
		return nil, fmt.Errorf("read hub state: %w", err)
	}
	var st HubState
	if err := json.Unmarshal(data, &st); err != nil {
		return nil, fmt.Errorf("parse hub state: %w", err)
	}
	if st.Sites == nil {
		st.Sites = make(map[string]*SiteState)
	}
	return &st, nil
}

// SaveState writes HubState atomically as formatted JSON to path.
func SaveState(path string, st *HubState) error {
	if st.Sites == nil {
		st.Sites = make(map[string]*SiteState)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal hub state: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

func (s *HubState) getOrCreateSite(siteID string) *SiteState {
	site, ok := s.Sites[siteID]
	if !ok {
		site = &SiteState{SiteID: siteID}
		s.Sites[siteID] = site
	}
	return site
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".miro-state-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
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
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	cleanup = false
	return nil
}
