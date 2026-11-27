package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/3nd3r1/miro/internal/bundle"
	"github.com/3nd3r1/miro/internal/status"
)

// Runner executes system commands (nix, nix-env, switch-to-configuration, systemctl).
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// ExecRunner executes commands via os/exec.
type ExecRunner struct{}

// Run executes name with args and captures stdout/stderr.
func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return "", fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, msg)
		}
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// HealthChecker probes the local inference server health endpoint.
type HealthChecker interface {
	Check(ctx context.Context, url string) error
}

// HTTPHealthChecker checks an HTTP endpoint and expects a 2xx response.
type HTTPHealthChecker struct {
	Client *http.Client
}

// Check performs an HTTP GET against url and succeeds on a 2xx status code.
func (c HTTPHealthChecker) Check(ctx context.Context, url string) error {
	client := c.Client
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("health endpoint returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Config configures an offline site agent reconciliation run.
type Config struct {
	SiteID         string
	InboundDir     string
	StatusOutDir   string
	StatePath      string
	ModelsDir      string
	SystemProfile  string
	HealthURL      string
	HealthTimeout  time.Duration
	HealthInterval time.Duration
	InferenceUnit  string
}

// Agent reconciles inbound air-gapped bundles on an offline site.
type Agent struct {
	Cfg    Config
	Runner Runner
	Health HealthChecker
	LogOut io.Writer
}

type verifiedCandidate struct {
	dir      string
	manifest *bundle.Manifest
}

// Reconcile scans InboundDir, ignores incomplete bundles, reports rejected bundles,
// applies valid unapplied bundles in sequence order, checks health, and writes status.
func (a *Agent) Reconcile(ctx context.Context) error {
	if strings.TrimSpace(a.Cfg.SiteID) == "" {
		return errors.New("site ID is required")
	}

	st, err := LoadAgentState(a.Cfg.StatePath, a.Cfg.SiteID)
	if err != nil {
		return err
	}

	candidates, err := discoverCandidates(a.Cfg.InboundDir)
	if err != nil {
		return err
	}

	valid, stateChanged := a.classifyCandidates(st, candidates)
	slices.SortFunc(valid, func(x, y verifiedCandidate) int {
		switch {
		case x.manifest.Sequence < y.manifest.Sequence:
			return -1
		case x.manifest.Sequence > y.manifest.Sequence:
			return 1
		default:
			return strings.Compare(x.manifest.BundleID, y.manifest.BundleID)
		}
	})

	var applyErr error
	for _, cand := range valid {
		if cand.manifest.Sequence <= st.LastAppliedSequence {
			continue
		}
		stateChanged = true
		if err := a.applyBundle(ctx, st, cand); err != nil {
			applyErr = err
			break
		}
	}

	if !stateChanged {
		return nil
	}

	health := a.waitForHealth(ctx)
	if err := a.persistAndReport(st, health); err != nil {
		return err
	}
	return applyErr
}

func (a *Agent) classifyCandidates(st *AgentState, dirs []string) ([]verifiedCandidate, bool) {
	var valid []verifiedCandidate
	changed := false

	for _, dir := range dirs {
		m, err := bundle.Verify(dir, a.Cfg.SiteID)
		if err == nil {
			valid = append(valid, verifiedCandidate{dir: dir, manifest: m})
			continue
		}
		if errors.Is(err, bundle.ErrIncomplete) {
			a.logf("deferring incomplete bundle %s: %v", filepath.Base(dir), err)
			continue
		}
		if a.recordRejection(st, dir, err) {
			changed = true
		}
	}
	return valid, changed
}

func (a *Agent) recordRejection(st *AgentState, dir string, verifyErr error) bool {
	key := filepath.Base(dir)
	reason := verifyErr.Error()
	if st.ReportedRejections[key] == reason {
		return false
	}

	var seq uint64
	bundleID := key
	if m, err := bundle.ReadManifest(filepath.Join(dir, bundle.ManifestFile)); err == nil {
		seq = m.Sequence
		bundleID = m.BundleID
	}

	st.ReportedRejections[key] = reason
	msg := fmt.Sprintf("rejected bundle %s: %s", bundleID, reason)
	a.logf("%s", msg)
	st.appendLog(msg)
	st.appendEvent(status.RolloutEvent{
		Sequence: seq,
		BundleID: bundleID,
		Outcome:  "rejected",
		Reason:   reason,
	})
	return true
}

func (a *Agent) applyBundle(ctx context.Context, st *AgentState, cand verifiedCandidate) error {
	m := cand.manifest
	a.recordLog(st, fmt.Sprintf("applying bundle %s (seq=%d system=%s)", m.BundleID, m.Sequence, m.TargetSystem))

	modelSHA, err := a.installModels(cand.dir, m.Models)
	if err != nil {
		a.recordFailure(st, m, modelSHA, fmt.Sprintf("install models: %v", err))
		return err
	}

	if err := a.importClosure(ctx, cand.dir, m); err != nil {
		a.recordFailure(st, m, modelSHA, fmt.Sprintf("import closure: %v", err))
		return err
	}

	if err := a.activateSystem(ctx, m.TargetSystem); err != nil {
		a.recordFailure(st, m, modelSHA, fmt.Sprintf("activate system: %v", err))
		return err
	}

	curSys, curGen := inspectSystemProfile(a.Cfg.SystemProfile, m.TargetSystem)
	st.LastAppliedSequence = m.Sequence
	st.CurrentSystem = curSys
	st.CurrentGeneration = curGen
	if modelSHA != "" {
		st.CurrentModelSHA256 = modelSHA
	}
	a.recordLog(st, fmt.Sprintf("applied bundle %s (gen=%d)", m.BundleID, curGen))
	st.appendEvent(status.RolloutEvent{
		Sequence:     m.Sequence,
		BundleID:     m.BundleID,
		TargetSystem: m.TargetSystem,
		ModelSHA256:  modelSHA,
		Outcome:      "applied",
	})
	return nil
}

func (a *Agent) installModels(bundleDir string, models []bundle.ModelArtifact) (string, error) {
	if len(models) == 0 {
		return "", nil
	}
	if err := os.MkdirAll(a.Cfg.ModelsDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir models dir: %w", err)
	}
	var lastSHA string
	for _, model := range models {
		dest := filepath.Join(a.Cfg.ModelsDir, model.SHA256)
		if existingSum, existingSize, err := bundle.HashFile(dest); err == nil &&
			existingSum == model.SHA256 && existingSize == model.Size {
			lastSHA = model.SHA256
			continue
		}
		src := filepath.Join(bundleDir, bundle.ModelsDir, model.SHA256)
		if err := copyModelAtomic(src, dest); err != nil {
			return "", fmt.Errorf("copy model %s: %w", model.SHA256, err)
		}
		lastSHA = model.SHA256
	}
	return lastSHA, nil
}

func (a *Agent) importClosure(ctx context.Context, bundleDir string, m *bundle.Manifest) error {
	if len(m.CacheFiles) == 0 {
		return nil
	}
	absDir, err := filepath.Abs(bundleDir)
	if err != nil {
		return fmt.Errorf("abs bundle dir: %w", err)
	}
	cacheURI := "file://" + filepath.Join(absDir, bundle.CacheDir)
	_, err = a.runner().Run(
		ctx,
		"nix",
		"--extra-experimental-features", "nix-command",
		"copy",
		"--no-check-sigs",
		"--from", cacheURI,
		m.TargetSystem,
	)
	return err
}

func (a *Agent) activateSystem(ctx context.Context, targetSystem string) error {
	if _, err := a.runner().Run(ctx, "nix-env", "-p", a.Cfg.SystemProfile, "--set", targetSystem); err != nil {
		return fmt.Errorf("set system profile: %w", err)
	}
	switchBin := filepath.Join(targetSystem, "bin", "switch-to-configuration")
	if _, err := a.runner().Run(ctx, switchBin, "switch"); err != nil {
		return fmt.Errorf("switch-to-configuration: %w", err)
	}
	if a.Cfg.InferenceUnit != "" {
		if _, err := a.runner().Run(ctx, "systemctl", "start", a.Cfg.InferenceUnit); err != nil {
			return fmt.Errorf("start %s: %w", a.Cfg.InferenceUnit, err)
		}
	}
	return nil
}

func (a *Agent) waitForHealth(ctx context.Context) status.Health {
	if strings.TrimSpace(a.Cfg.HealthURL) == "" {
		return status.Health{Healthy: true, Message: "health check disabled"}
	}
	timeout := a.Cfg.HealthTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	interval := a.Cfg.HealthInterval
	if interval <= 0 {
		interval = time.Second
	}

	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var lastErr error
	for {
		if err := a.healthChecker().Check(checkCtx, a.Cfg.HealthURL); err == nil {
			return status.Health{Healthy: true, Message: "ok"}
		} else {
			lastErr = err
		}

		timer := time.NewTimer(interval)
		select {
		case <-checkCtx.Done():
			timer.Stop()
			msg := checkCtx.Err().Error()
			if lastErr != nil {
				msg = lastErr.Error()
			}
			return status.Health{Healthy: false, Message: msg}
		case <-timer.C:
		}
	}
}

func (a *Agent) persistAndReport(st *AgentState, health status.Health) error {
	if st.CurrentSystem == "" {
		curSys, curGen := inspectSystemProfile(a.Cfg.SystemProfile, "")
		st.CurrentSystem = curSys
		st.CurrentGeneration = curGen
	}
	if err := SaveAgentState(a.Cfg.StatePath, st); err != nil {
		return err
	}
	sb := &status.Bundle{
		SiteID:              st.SiteID,
		LastAppliedSequence: st.LastAppliedSequence,
		CurrentSystem:       st.CurrentSystem,
		CurrentGeneration:   st.CurrentGeneration,
		CurrentModelSHA256:  st.CurrentModelSHA256,
		Health:              health,
		RolloutEvents:       st.RolloutEvents,
		AgentLogTail:        st.AgentLogTail,
	}
	return status.WriteDir(a.Cfg.StatusOutDir, sb)
}

func (a *Agent) recordFailure(st *AgentState, m *bundle.Manifest, modelSHA, reason string) {
	a.recordLog(st, fmt.Sprintf("failed bundle %s: %s", m.BundleID, reason))
	st.appendEvent(status.RolloutEvent{
		Sequence:     m.Sequence,
		BundleID:     m.BundleID,
		TargetSystem: m.TargetSystem,
		ModelSHA256:  modelSHA,
		Outcome:      "failed",
		Reason:       reason,
	})
}

func (a *Agent) recordLog(st *AgentState, msg string) {
	a.logf("%s", msg)
	st.appendLog(msg)
}

func (a *Agent) logf(format string, args ...any) {
	if a.LogOut != nil {
		fmt.Fprintf(a.LogOut, "miro-agent: "+format+"\n", args...)
	}
}

func (a *Agent) runner() Runner {
	if a.Runner != nil {
		return a.Runner
	}
	return ExecRunner{}
}

func (a *Agent) healthChecker() HealthChecker {
	if a.Health != nil {
		return a.Health
	}
	return HTTPHealthChecker{}
}

func discoverCandidates(inboundDir string) ([]string, error) {
	entries, err := os.ReadDir(inboundDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read inbound dir: %w", err)
	}

	for _, e := range entries {
		name := e.Name()
		if name == bundle.ManifestFile || name == bundle.CacheDir || name == bundle.ModelsDir {
			return []string{inboundDir}, nil
		}
	}

	var dirs []string
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, filepath.Join(inboundDir, e.Name()))
	}
	slices.Sort(dirs)
	return dirs, nil
}

func inspectSystemProfile(profilePath, fallbackSystem string) (string, int) {
	gen := parseGenerationFromProfile(profilePath)
	if resolved, err := filepath.EvalSymlinks(profilePath); err == nil && resolved != "" {
		return resolved, gen
	}
	return fallbackSystem, gen
}

func parseGenerationFromProfile(profilePath string) int {
	target, err := os.Readlink(profilePath)
	if err != nil {
		return 0
	}
	trimmed, ok := strings.CutPrefix(filepath.Base(target), "system-")
	if !ok {
		return 0
	}
	numStr, ok := strings.CutSuffix(trimmed, "-link")
	if !ok {
		return 0
	}
	n, err := strconv.Atoi(numStr)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func copyModelAtomic(srcPath, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	tmp, err := os.CreateTemp(filepath.Dir(dstPath), ".model-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
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
	if err := os.Rename(tmpName, dstPath); err != nil {
		return err
	}
	cleanup = false
	return nil
}
