package main

import (
	"bytes"
	"context"
	"encoding/json"
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

func populateFakeCache(t *testing.T, toArg string) {
	t.Helper()
	cachePath := strings.TrimPrefix(toArg, "file://")
	if idx := strings.IndexByte(cachePath, '?'); idx >= 0 {
		cachePath = cachePath[:idx]
	}
	if err := os.MkdirAll(filepath.Join(cachePath, "nar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cachePath, "nix-cache-info"), []byte("StoreDir: /nix/store\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cachePath, "nar", "test.nar"), []byte("fake-nar-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestStateLoadSaveRoundTrip(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "nested", "state.json")

	st, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState on missing file: %v", err)
	}
	if len(st.Sites) != 0 {
		t.Fatalf("expected empty sites, got %v", st.Sites)
	}

	site := st.getOrCreateSite("site-a")
	site.LastExportedSequence = 2
	site.Desired = &DesiredState{
		Sequence:   2,
		BundleID:   "site-a-seq-0002",
		SystemPath: "/nix/store/aaaa-nixos-system-site-a",
	}

	if err := SaveState(statePath, st); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	got, err := LoadState(statePath)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	if got.Sites["site-a"].LastExportedSequence != 2 {
		t.Fatalf("unexpected sequence: %+v", got.Sites["site-a"])
	}
}

func TestHubBuildExportImportAndStatus(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "hub-state.json")
	modelSrc := filepath.Join(tmp, "tiny.gguf")
	if err := os.WriteFile(modelSrc, []byte("gguf-model-data-v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	wantModelSum, _, err := bundle.HashFile(modelSrc)
	if err != nil {
		t.Fatal(err)
	}

	const sysV1 = "/nix/store/1111-nixos-system-site-a"
	const sysV2 = "/nix/store/2222-nixos-system-site-a"

	runner := &fakeRunner{
		fn: func(_ string, args []string) (string, error) {
			if slicesContains(args, "build") {
				return sysV1 + "\n", nil
			}
			for i, a := range args {
				if a == "--to" && i+1 < len(args) {
					populateFakeCache(t, args[i+1])
				}
			}
			return "", nil
		},
	}

	fixedTime := time.Date(2026, 11, 27, 12, 0, 0, 0, time.UTC)
	hub := &Hub{
		StatePath: statePath,
		Runner:    runner,
		Now:       func() time.Time { return fixedTime },
	}

	// 1. Test Build
	built, err := hub.Build(context.Background(), ".", "site-a")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if built != sysV1 {
		t.Fatalf("Build got %q, want %q", built, sysV1)
	}

	// 2. Test Export (sequence 1, using built system + model)
	out1 := filepath.Join(tmp, "bundle-1")
	m1, err := hub.Export(context.Background(), ExportOptions{
		Site:        "site-a",
		OutDir:      out1,
		ModelPath:   modelSrc,
		Compression: "none",
	})
	if err != nil {
		t.Fatalf("Export 1: %v", err)
	}
	if m1.Sequence != 1 || m1.BundleID != "site-a-seq-0001" || m1.TargetSystem != sysV1 {
		t.Fatalf("unexpected manifest 1: %+v", m1)
	}
	if _, err := bundle.Verify(out1, "site-a"); err != nil {
		t.Fatalf("Verify bundle 1: %v", err)
	}

	// Confirm model weights are stored content-addressed under models/<sha256>.
	if _, err := os.Stat(filepath.Join(out1, bundle.ModelsDir, wantModelSum)); err != nil {
		t.Fatalf("expected model at models/%s: %v", wantModelSum, err)
	}

	// 3. Test Export (sequence 2, using pre-built --system)
	out2 := filepath.Join(tmp, "bundle-2")
	m2, err := hub.Export(context.Background(), ExportOptions{
		Site:        "site-a",
		OutDir:      out2,
		SystemPath:  sysV2,
		ModelPath:   modelSrc,
		Compression: "none",
	})
	if err != nil {
		t.Fatalf("Export 2: %v", err)
	}
	if m2.Sequence != 2 || m2.TargetSystem != sysV2 {
		t.Fatalf("unexpected manifest 2: %+v", m2)
	}

	// Status before import should list both pending bundles and actual: none.
	var beforeBuf bytes.Buffer
	if err := hub.WriteStatus(&beforeBuf, false); err != nil {
		t.Fatalf("WriteStatus before: %v", err)
	}
	beforeStr := beforeBuf.String()
	for _, want := range []string{
		"site: site-a",
		"desired: seq=2 system=" + sysV2,
		"actual:  none",
		"site-a-seq-0001(seq=1), site-a-seq-0002(seq=2)",
	} {
		if !strings.Contains(beforeStr, want) {
			t.Fatalf("status output missing %q:\n%s", want, beforeStr)
		}
	}

	// 4. Import status for seq=2.
	statusDir := filepath.Join(tmp, "status-in")
	sb := &status.Bundle{
		SiteID:              "site-a",
		LastAppliedSequence: 2,
		CurrentSystem:       sysV2,
		CurrentGeneration:   4,
		CurrentModelSHA256:  wantModelSum,
		Health:              status.Health{Healthy: true, Message: "ok"},
	}
	if err := status.WriteDir(statusDir, sb); err != nil {
		t.Fatalf("status.WriteDir: %v", err)
	}
	if _, err := hub.ImportStatus(statusDir); err != nil {
		t.Fatalf("ImportStatus: %v", err)
	}

	// Status after import should show actual seq=2, healthy, pending: none.
	var afterBuf bytes.Buffer
	if err := hub.WriteStatus(&afterBuf, false); err != nil {
		t.Fatalf("WriteStatus after: %v", err)
	}
	afterStr := afterBuf.String()
	for _, want := range []string{
		"actual:  seq=2 gen=4 system=" + sysV2 + " model=" + wantModelSum + " health=healthy (ok)",
		"pending: none",
	} {
		if !strings.Contains(afterStr, want) {
			t.Fatalf("status output missing %q:\n%s", want, afterStr)
		}
	}

	// JSON status output should deserialize into HubState.
	var jsonBuf bytes.Buffer
	if err := hub.WriteStatus(&jsonBuf, true); err != nil {
		t.Fatalf("WriteStatus json: %v", err)
	}
	var decoded HubState
	if err := json.Unmarshal(jsonBuf.Bytes(), &decoded); err != nil {
		t.Fatalf("decode status json: %v", err)
	}
	if decoded.Sites["site-a"].Actual.CurrentGeneration != 4 {
		t.Fatalf("unexpected decoded generation: %+v", decoded.Sites["site-a"].Actual)
	}
}

func TestCLISubcommands(t *testing.T) {
	tmp := t.TempDir()
	statePath := filepath.Join(tmp, "state.json")
	outDir := filepath.Join(tmp, "out")

	runner := &fakeRunner{
		fn: func(_ string, args []string) (string, error) {
			if slicesContains(args, "build") {
				return "/nix/store/aaaa-nixos-system-site-a\n", nil
			}
			for i, a := range args {
				if a == "--to" && i+1 < len(args) {
					populateFakeCache(t, args[i+1])
				}
			}
			return "", nil
		},
	}

	var stdout, stderr bytes.Buffer
	if code := runWithRunner(context.Background(), []string{"build", "site-a"}, runner, &stdout, &stderr); code != 0 {
		t.Fatalf("build failed (%d): %s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "/nix/store/aaaa-nixos-system-site-a" {
		t.Fatalf("unexpected build output: %q", stdout.String())
	}

	stdout.Reset()
	stderr.Reset()
	code := runWithRunner(
		context.Background(),
		[]string{"export", "--state", statePath, "--system", "/nix/store/aaaa-nixos-system-site-a", "site-a", outDir},
		runner,
		&stdout,
		&stderr,
	)
	if code != 0 {
		t.Fatalf("export failed (%d): %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "site-a-seq-0001") {
		t.Fatalf("unexpected export output: %q", stdout.String())
	}

	statusDir := filepath.Join(tmp, "status")
	if err := status.WriteDir(statusDir, &status.Bundle{
		SiteID:              "site-a",
		LastAppliedSequence: 1,
		CurrentSystem:       "/nix/store/aaaa-nixos-system-site-a",
		CurrentGeneration:   2,
		Health:              status.Health{Healthy: true},
	}); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	if code := runWithRunner(context.Background(), []string{"import-status", "--state", statePath, statusDir}, runner, &stdout, &stderr); code != 0 {
		t.Fatalf("import-status failed (%d): %s", code, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if code := runWithRunner(context.Background(), []string{"status", "--state", statePath}, runner, &stdout, &stderr); code != 0 {
		t.Fatalf("status failed (%d): %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "health=healthy") {
		t.Fatalf("unexpected status output: %q", stdout.String())
	}
}

func TestHubRunnerErrors(t *testing.T) {
	hub := &Hub{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
		Runner: &fakeRunner{
			fn: func(_ string, _ []string) (string, error) {
				return "", errors.New("nix boom")
			},
		},
	}
	if _, err := hub.Build(context.Background(), ".", "site-a"); err == nil {
		t.Fatalf("expected error from Build")
	}
	if _, err := hub.ImportStatus(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatalf("expected error from ImportStatus on missing dir")
	}
}

func slicesContains(items []string, target string) bool {
	for _, it := range items {
		if it == target {
			return true
		}
	}
	return false
}
