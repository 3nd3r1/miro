package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/3nd3r1/miro/internal/bundle"
	"github.com/3nd3r1/miro/internal/status"
)

// Runner executes external commands such as nix build and nix copy.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

// ExecRunner runs commands via os/exec.
type ExecRunner struct{}

// Run executes name with args and returns trimmed stdout or a descriptive error including stderr.
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

// Hub implements the hub CLI operations against a local JSON state file.
type Hub struct {
	StatePath string
	Runner    Runner
	Now       func() time.Time
}

func (h *Hub) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *Hub) runner() Runner {
	if h.Runner != nil {
		return h.Runner
	}
	return ExecRunner{}
}

// Build builds a site's NixOS toplevel closure and returns its /nix/store path.
func (h *Hub) Build(ctx context.Context, flakeRef, site string) (string, error) {
	if strings.TrimSpace(site) == "" {
		return "", errors.New("site name is required")
	}
	if strings.TrimSpace(flakeRef) == "" {
		flakeRef = "."
	}
	installable := fmt.Sprintf("%s#nixosConfigurations.%s.config.system.build.toplevel", flakeRef, site)
	out, err := h.runner().Run(
		ctx,
		"nix",
		"--extra-experimental-features", "nix-command flakes",
		"build",
		"--no-link",
		"--print-out-paths",
		installable,
	)
	if err != nil {
		return "", fmt.Errorf("build site %q: %w", site, err)
	}
	lines := strings.Fields(out)
	if len(lines) == 0 {
		return "", fmt.Errorf("build site %q: nix build returned empty output", site)
	}
	return lines[len(lines)-1], nil
}

// ExportOptions configures an outbound bundle export.
type ExportOptions struct {
	Site        string
	OutDir      string
	FlakeRef    string
	SystemPath  string
	BaseSystem  string
	ModelPath   string
	ModelName   string
	BundleID    string
	Compression string
}

// Export builds or takes a pre-built system closure, copies its Nix cache and optional model
// weights into opts.OutDir, writes manifest.json, and records the export in hub state.
func (h *Hub) Export(ctx context.Context, opts ExportOptions) (*bundle.Manifest, error) {
	if strings.TrimSpace(opts.Site) == "" {
		return nil, errors.New("site name is required")
	}
	if strings.TrimSpace(opts.OutDir) == "" {
		return nil, errors.New("output directory is required")
	}

	systemPath, err := h.resolveSystemPath(ctx, opts)
	if err != nil {
		return nil, err
	}

	st, err := LoadState(h.StatePath)
	if err != nil {
		return nil, err
	}
	site := st.getOrCreateSite(opts.Site)

	absOutDir, err := filepath.Abs(opts.OutDir)
	if err != nil {
		return nil, fmt.Errorf("resolve output directory: %w", err)
	}
	cacheFiles, err := h.exportNixCache(ctx, absOutDir, systemPath, opts.Compression)
	if err != nil {
		return nil, err
	}
	models, modelSHA, err := exportModel(absOutDir, opts.ModelPath, opts.ModelName)
	if err != nil {
		return nil, err
	}

	m := h.buildManifest(site, opts, systemPath, models, cacheFiles)
	if err := bundle.WriteManifest(filepath.Join(absOutDir, bundle.ManifestFile), m); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	recordExport(site, m, modelSHA, absOutDir)
	if err := SaveState(h.StatePath, st); err != nil {
		return nil, err
	}
	return m, nil
}

func (h *Hub) resolveSystemPath(ctx context.Context, opts ExportOptions) (string, error) {
	if strings.TrimSpace(opts.SystemPath) != "" {
		return strings.TrimSpace(opts.SystemPath), nil
	}
	return h.Build(ctx, opts.FlakeRef, opts.Site)
}

func (h *Hub) exportNixCache(ctx context.Context, absOutDir, systemPath, compression string) ([]bundle.FileChecksum, error) {
	cacheDir := filepath.Join(absOutDir, bundle.CacheDir)
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return nil, fmt.Errorf("create cache dir: %w", err)
	}
	dest := "file://" + cacheDir
	if compression != "" {
		dest += "?compression=" + compression
	}
	_, err := h.runner().Run(
		ctx,
		"nix",
		"--extra-experimental-features", "nix-command",
		"copy",
		"--no-check-sigs",
		"--to", dest,
		systemPath,
	)
	if err != nil {
		return nil, fmt.Errorf("copy closure to bundle cache: %w", err)
	}
	files, err := bundle.ScanCacheDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("checksum cache dir: %w", err)
	}
	return files, nil
}

func exportModel(absOutDir, modelPath, modelName string) ([]bundle.ModelArtifact, string, error) {
	if strings.TrimSpace(modelPath) == "" {
		return nil, "", nil
	}
	sum, size, err := bundle.HashFile(modelPath)
	if err != nil {
		return nil, "", fmt.Errorf("hash model %q: %w", modelPath, err)
	}
	destPath := filepath.Join(absOutDir, bundle.ModelsDir, sum)
	if err := copyFileAtomic(modelPath, destPath, 0o644); err != nil {
		return nil, "", fmt.Errorf("copy model to bundle: %w", err)
	}
	name := strings.TrimSpace(modelName)
	if name == "" {
		name = filepath.Base(modelPath)
	}
	return []bundle.ModelArtifact{{Name: name, SHA256: sum, Size: size}}, sum, nil
}

func (h *Hub) buildManifest(
	site *SiteState,
	opts ExportOptions,
	systemPath string,
	models []bundle.ModelArtifact,
	cacheFiles []bundle.FileChecksum,
) *bundle.Manifest {
	seq := site.LastExportedSequence + 1
	bundleID := strings.TrimSpace(opts.BundleID)
	if bundleID == "" {
		bundleID = fmt.Sprintf("%s-seq-%04d", opts.Site, seq)
	}
	baseSystem := strings.TrimSpace(opts.BaseSystem)
	if baseSystem == "" && site.Actual != nil {
		baseSystem = site.Actual.CurrentSystem
	}
	return &bundle.Manifest{
		BundleID:     bundleID,
		SiteID:       opts.Site,
		Sequence:     seq,
		TargetSystem: systemPath,
		BaseSystem:   baseSystem,
		Models:       models,
		CacheFiles:   cacheFiles,
		CreatedAt:    h.now().UTC().Format(time.RFC3339),
	}
}

func recordExport(site *SiteState, m *bundle.Manifest, modelSHA, absOutDir string) {
	site.LastExportedSequence = m.Sequence
	site.Desired = &DesiredState{
		Sequence:    m.Sequence,
		BundleID:    m.BundleID,
		SystemPath:  m.TargetSystem,
		ModelSHA256: modelSHA,
		ExportedAt:  m.CreatedAt,
	}
	site.PendingBundles = append(site.PendingBundles, ExportedBundle{
		BundleID:    m.BundleID,
		Sequence:    m.Sequence,
		SystemPath:  m.TargetSystem,
		ModelSHA256: modelSHA,
		OutputDir:   absOutDir,
		ExportedAt:  m.CreatedAt,
	})
}

// ImportStatus reads and validates a status bundle from dirOrFile and updates hub state.
func (h *Hub) ImportStatus(dirOrFile string) (*status.Bundle, error) {
	if strings.TrimSpace(dirOrFile) == "" {
		return nil, errors.New("status path is required")
	}
	sb, err := readStatusBundle(dirOrFile)
	if err != nil {
		return nil, err
	}

	st, err := LoadState(h.StatePath)
	if err != nil {
		return nil, err
	}
	site := st.getOrCreateSite(sb.SiteID)
	site.Actual = &ActualState{
		LastAppliedSequence: sb.LastAppliedSequence,
		CurrentSystem:       sb.CurrentSystem,
		CurrentGeneration:   sb.CurrentGeneration,
		CurrentModelSHA256:  sb.CurrentModelSHA256,
		Health:              sb.Health,
		RolloutEvents:       sb.RolloutEvents,
		AgentLogTail:        sb.AgentLogTail,
		ImportedAt:          h.now().UTC().Format(time.RFC3339),
	}
	site.PendingBundles = filterPendingBundles(site.PendingBundles, sb.LastAppliedSequence)

	if err := SaveState(h.StatePath, st); err != nil {
		return nil, err
	}
	return sb, nil
}

func readStatusBundle(dirOrFile string) (*status.Bundle, error) {
	info, err := os.Stat(dirOrFile)
	if err != nil {
		return nil, fmt.Errorf("stat status path: %w", err)
	}
	if info.IsDir() {
		return status.ReadDir(dirOrFile)
	}
	return status.ReadFile(dirOrFile)
}

func filterPendingBundles(pending []ExportedBundle, appliedSeq uint64) []ExportedBundle {
	var remaining []ExportedBundle
	for _, b := range pending {
		if b.Sequence > appliedSeq {
			remaining = append(remaining, b)
		}
	}
	return remaining
}

// WriteStatus formats desired vs actual state for all sites to w.
func (h *Hub) WriteStatus(w io.Writer, asJSON bool) error {
	st, err := LoadState(h.StatePath)
	if err != nil {
		return err
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(st)
	}
	if len(st.Sites) == 0 {
		fmt.Fprintln(w, "no sites tracked")
		return nil
	}

	siteIDs := make([]string, 0, len(st.Sites))
	for id := range st.Sites {
		siteIDs = append(siteIDs, id)
	}
	slices.Sort(siteIDs)

	for i, id := range siteIDs {
		if i > 0 {
			fmt.Fprintln(w)
		}
		writeSiteStatus(w, st.Sites[id])
	}
	return nil
}

func writeSiteStatus(w io.Writer, site *SiteState) {
	fmt.Fprintf(w, "site: %s\n", site.SiteID)
	if site.Desired != nil {
		fmt.Fprintf(w, "  desired: seq=%d system=%s model=%s\n",
			site.Desired.Sequence,
			site.Desired.SystemPath,
			emptyDash(site.Desired.ModelSHA256),
		)
	} else {
		fmt.Fprintln(w, "  desired: none")
	}

	if site.Actual != nil {
		healthStr := "unhealthy"
		if site.Actual.Health.Healthy {
			healthStr = "healthy"
		}
		if site.Actual.Health.Message != "" {
			healthStr += " (" + site.Actual.Health.Message + ")"
		}
		fmt.Fprintf(w, "  actual:  seq=%d gen=%d system=%s model=%s health=%s\n",
			site.Actual.LastAppliedSequence,
			site.Actual.CurrentGeneration,
			emptyDash(site.Actual.CurrentSystem),
			emptyDash(site.Actual.CurrentModelSHA256),
			healthStr,
		)
	} else {
		fmt.Fprintln(w, "  actual:  none")
	}

	if len(site.PendingBundles) == 0 {
		fmt.Fprintln(w, "  pending: none")
		return
	}
	pendingParts := make([]string, 0, len(site.PendingBundles))
	for _, b := range site.PendingBundles {
		pendingParts = append(pendingParts, fmt.Sprintf("%s(seq=%d)", b.BundleID, b.Sequence))
	}
	fmt.Fprintf(w, "  pending: %s\n", strings.Join(pendingParts, ", "))
}

func emptyDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func copyFileAtomic(srcPath, dstPath string, perm os.FileMode) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source %s: %w", srcPath, err)
	}
	defer func() { _ = src.Close() }()

	dir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".copy-tmp-*")
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

	if _, err := io.Copy(tmp, src); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("copy bytes: %w", err)
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
	if err := os.Rename(tmpName, dstPath); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	cleanup = false
	return nil
}
