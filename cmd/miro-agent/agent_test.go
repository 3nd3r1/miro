package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/3nd3r1/miro/internal/bundle"
	"github.com/3nd3r1/miro/internal/status"
)

type fakeRunner struct {
	calls [][]string
	fn    func(name string, args []string) (string, error)
}

func (f *fakeRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)
	if f.fn != nil {
		return f.fn(name, args)
	}
	return "", nil
}

type fakeHealth struct {
	errs  []error
	calls int
}

func (f *fakeHealth) Check(_ context.Context, _ string) error {
	idx := f.calls
	f.calls++
	if idx < len(f.errs) {
		return f.errs[idx]
	}
	return nil
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func writeTestBundle(t *testing.T, dir, siteID string, seq uint64, targetSystem string, modelData []byte) string {
	t.Helper()
	modelSum := sha256Hex(modelData)

	modelPath := filepath.Join(dir, bundle.ModelsDir, modelSum)
	if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, modelData, 0o644); err != nil {
		t.Fatal(err)
	}

	cacheDir := filepath.Join(dir, bundle.CacheDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "nar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "nix-cache-info"), []byte("StoreDir: /nix/store\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "nar", "closure.nar"), []byte("closure-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	cacheFiles, err := bundle.ScanCacheDir(cacheDir)
	if err != nil {
		t.Fatal(err)
	}

	m := &bundle.Manifest{
		BundleID:     siteID + "-seq-" + string(rune('0'+seq)),
		SiteID:       siteID,
		Sequence:     seq,
		TargetSystem: targetSystem,
		Models: []bundle.ModelArtifact{
			{Name: "tiny.gguf", SHA256: modelSum, Size: int64(len(modelData))},
		},
		CacheFiles: cacheFiles,
	}
	if err := bundle.WriteManifest(filepath.Join(dir, bundle.ManifestFile), m); err != nil {
		t.Fatal(err)
	}
	return modelSum
}

func setupProfileSymlink(t *testing.T, profileDir, targetSystem string, gen int) {
	t.Helper()
	// Create a fake store path directory so EvalSymlinks resolves it cleanly.
	if err := os.MkdirAll(targetSystem, 0o755); err != nil {
		t.Fatal(err)
	}
	genLink := filepath.Join(profileDir, "system-"+string(rune('0'+gen))+"-link")
	_ = os.Remove(genLink)
	if err := os.Symlink(targetSystem, genLink); err != nil {
		t.Fatal(err)
	}
	profilePath := filepath.Join(profileDir, "system")
	_ = os.Remove(profilePath)
	if err := os.Symlink(filepath.Base(genLink), profilePath); err != nil {
		t.Fatal(err)
	}
}

func TestAgentPartialDeferralThenApplyAndIdempotentRerun(t *testing.T) {
	tmp := t.TempDir()
	inboundDir := filepath.Join(tmp, "inbound")
	statusOutDir := filepath.Join(tmp, "status-out")
	statePath := filepath.Join(tmp, "agent-state.json")
	modelsDir := filepath.Join(tmp, "models")
	profileDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	targetSystem := filepath.Join(tmp, "nix-store-sys-v2")
	bundleDir := filepath.Join(inboundDir, "site-a-seq-1")
	modelBytes := []byte("gguf-weights-payload-v2")
	modelSum := writeTestBundle(t, bundleDir, "site-a", 1, targetSystem, modelBytes)

	// Simulate an incomplete copy by truncating the model file in the inbound bundle.
	modelInBundle := filepath.Join(bundleDir, bundle.ModelsDir, modelSum)
	if err := os.WriteFile(modelInBundle, []byte("short"), 0o644); err != nil {
		t.Fatal(err)
	}

	profilePath := filepath.Join(profileDir, "system")
	runner := &fakeRunner{
		fn: func(name string, args []string) (string, error) {
			if name == "nix-env" {
				setupProfileSymlink(t, profileDir, targetSystem, 2)
			}
			return "", nil
		},
	}
	health := &fakeHealth{errs: []error{errors.New("starting"), nil}}
	var logBuf bytes.Buffer

	agent := &Agent{
		Cfg: Config{
			SiteID:         "site-a",
			InboundDir:     inboundDir,
			StatusOutDir:   statusOutDir,
			StatePath:      statePath,
			ModelsDir:      modelsDir,
			SystemProfile:  profilePath,
			HealthURL:      "http://127.0.0.1:8080/health",
			HealthTimeout:  2 * time.Second,
			HealthInterval: 5 * time.Millisecond,
			InferenceUnit:  "llama-cpp.service",
		},
		Runner: runner,
		Health: health,
		LogOut: &logBuf,
	}

	// Run 1: partial bundle must be ignored with no commands executed and no state saved.
	if err := agent.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile partial: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("expected 0 commands on partial bundle, got %v", runner.calls)
	}
	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no persisted state after partial deferral, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(statusOutDir, status.StatusFile)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no status file after partial deferral, stat err=%v", err)
	}

	// Complete the copy of the model weights.
	if err := os.WriteFile(modelInBundle, modelBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	// Run 2: complete bundle should be installed, imported, activated, health-checked, and reported.
	if err := agent.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile complete: %v", err)
	}
	if len(runner.calls) != 4 {
		t.Fatalf("expected 4 commands (nix copy, nix-env, switch, systemctl), got %d: %v", len(runner.calls), runner.calls)
	}
	if runner.calls[0][0] != "nix" || runner.calls[1][0] != "nix-env" ||
		!strings.HasSuffix(runner.calls[2][0], "switch-to-configuration") ||
		runner.calls[3][0] != "systemctl" {
		t.Fatalf("unexpected command sequence: %v", runner.calls)
	}

	// Verify model was copied atomically into modelsDir/<sha256>.
	gotModelBytes, err := os.ReadFile(filepath.Join(modelsDir, modelSum))
	if err != nil {
		t.Fatalf("read installed model: %v", err)
	}
	if string(gotModelBytes) != string(modelBytes) {
		t.Fatalf("installed model mismatch")
	}

	// Verify status bundle was emitted cleanly.
	sb, err := status.ReadDir(statusOutDir)
	if err != nil {
		t.Fatalf("status.ReadDir: %v", err)
	}
	if sb.SiteID != "site-a" || sb.LastAppliedSequence != 1 || sb.CurrentGeneration != 2 ||
		sb.CurrentSystem != targetSystem || sb.CurrentModelSHA256 != modelSum || !sb.Health.Healthy {
		t.Fatalf("unexpected status bundle: %+v", sb)
	}
	if len(sb.RolloutEvents) != 1 || sb.RolloutEvents[0].Outcome != "applied" {
		t.Fatalf("unexpected rollout events: %+v", sb.RolloutEvents)
	}
	if len(sb.AgentLogTail) == 0 {
		t.Fatalf("expected non-empty agent log tail")
	}

	// Run 3: re-running with the same bundle still in inbound/ must be a no-op.
	callsBefore := len(runner.calls)
	if err := agent.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile rerun: %v", err)
	}
	if len(runner.calls) != callsBefore {
		t.Fatalf("expected idempotent rerun with no new commands, got %v", runner.calls[callsBefore:])
	}
}

func TestAgentWrongSiteRejectionReported(t *testing.T) {
	tmp := t.TempDir()
	inboundDir := filepath.Join(tmp, "inbound")
	statusOutDir := filepath.Join(tmp, "status-out")
	statePath := filepath.Join(tmp, "agent-state.json")
	modelsDir := filepath.Join(tmp, "models")

	// Drop a valid bundle addressed to site-b into site-a's inbound dir.
	bundleDir := filepath.Join(inboundDir, "site-b-seq-1")
	writeTestBundle(t, bundleDir, "site-b", 1, "/nix/store/sys-b", []byte("model-b"))

	runner := &fakeRunner{}
	agent := &Agent{
		Cfg: Config{
			SiteID:         "site-a",
			InboundDir:     inboundDir,
			StatusOutDir:   statusOutDir,
			StatePath:      statePath,
			ModelsDir:      modelsDir,
			SystemProfile:  filepath.Join(tmp, "missing-profile"),
			HealthURL:      "http://127.0.0.1:8080/health",
			HealthTimeout:  100 * time.Millisecond,
			HealthInterval: 5 * time.Millisecond,
		},
		Runner: runner,
		Health: &fakeHealth{},
	}

	if err := agent.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile wrong-site: %v", err)
	}
	if len(runner.calls) != 0 {
		t.Fatalf("expected 0 commands for wrong-site bundle, got %v", runner.calls)
	}

	sb, err := status.ReadDir(statusOutDir)
	if err != nil {
		t.Fatalf("status.ReadDir: %v", err)
	}
	if sb.LastAppliedSequence != 0 {
		t.Fatalf("expected sequence 0 after rejection, got %d", sb.LastAppliedSequence)
	}
	if len(sb.RolloutEvents) != 1 || sb.RolloutEvents[0].Outcome != "rejected" {
		t.Fatalf("expected 1 rejected rollout event, got %+v", sb.RolloutEvents)
	}
	if !strings.Contains(sb.RolloutEvents[0].Reason, "site mismatch") {
		t.Fatalf("expected site mismatch reason, got %q", sb.RolloutEvents[0].Reason)
	}

	// Re-running should not duplicate the same rejection event.
	if err := agent.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile second pass: %v", err)
	}
	sb2, err := status.ReadDir(statusOutDir)
	if err != nil {
		t.Fatalf("status.ReadDir second pass: %v", err)
	}
	if len(sb2.RolloutEvents) != 1 {
		t.Fatalf("expected no duplicate rejection events, got %+v", sb2.RolloutEvents)
	}
}

func TestAgentCLIReconcileFlatInbound(t *testing.T) {
	tmp := t.TempDir()
	inboundDir := filepath.Join(tmp, "inbound")
	statusOutDir := filepath.Join(tmp, "status-out")
	statePath := filepath.Join(tmp, "state.json")
	modelsDir := filepath.Join(tmp, "models")
	profileDir := filepath.Join(tmp, "profiles")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}

	targetSystem := filepath.Join(tmp, "nix-store-flat-v1")
	writeTestBundle(t, inboundDir, "site-a", 1, targetSystem, []byte("flat-model"))

	runner := &fakeRunner{
		fn: func(name string, _ []string) (string, error) {
			if name == "nix-env" {
				setupProfileSymlink(t, profileDir, targetSystem, 1)
			}
			return "", nil
		},
	}

	var stdout, stderr bytes.Buffer
	code := runWithDeps(
		context.Background(),
		[]string{
			"reconcile",
			"--site", "site-a",
			"--inbound", inboundDir,
			"--status-out", statusOutDir,
			"--state", statePath,
			"--models-dir", modelsDir,
			"--system-profile", filepath.Join(profileDir, "system"),
			"--health-timeout", "50ms",
			"--health-interval", "5ms",
			"--inference-unit", "",
		},
		runner,
		&fakeHealth{},
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("reconcile CLI failed (%d): %s", code, stderr.String())
	}

	sb, err := status.ReadDir(statusOutDir)
	if err != nil {
		t.Fatalf("status.ReadDir: %v", err)
	}
	if sb.LastAppliedSequence != 1 || sb.CurrentSystem != targetSystem || !sb.Health.Healthy {
		t.Fatalf("unexpected flat status: %+v", sb)
	}
}
