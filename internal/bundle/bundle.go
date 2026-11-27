package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	// ManifestFile is the filename of the bundle manifest at the bundle root.
	ManifestFile = "manifest.json"
	// CacheDir is the subdirectory holding the Nix binary cache closure.
	CacheDir = "cache"
	// ModelsDir is the subdirectory holding content-addressed model weights.
	ModelsDir = "models"
)

var (
	// ErrIncomplete indicates a bundle is missing files or has truncated files
	// (for example, an in-progress directory copy) and should be ignored until complete.
	ErrIncomplete = errors.New("bundle incomplete")
	// ErrCorrupt indicates a bundle file failed checksum or size validation or contains malformed JSON.
	ErrCorrupt = errors.New("bundle corrupt")
	// ErrSiteMismatch indicates the bundle manifest targets a different site ID.
	ErrSiteMismatch = errors.New("bundle site mismatch")
	// ErrInvalidManifest indicates the manifest JSON parsed but failed schema validation.
	ErrInvalidManifest = errors.New("invalid bundle manifest")
)

// Manifest describes an outbound air-gapped deployment bundle.
type Manifest struct {
	BundleID     string          `json:"bundle_id"`
	SiteID       string          `json:"site_id"`
	Sequence     uint64          `json:"sequence"`
	TargetSystem string          `json:"target_system"`
	BaseSystem   string          `json:"base_system,omitempty"`
	Models       []ModelArtifact `json:"models,omitempty"`
	CacheFiles   []FileChecksum  `json:"cache_files,omitempty"`
	EvalSuite    string          `json:"eval_suite,omitempty"`
	CreatedAt    string          `json:"created_at,omitempty"`
}

// ModelArtifact describes a content-addressed GGUF model stored under models/<sha256>.
type ModelArtifact struct {
	Name   string `json:"name,omitempty"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// FileChecksum records the relative path, SHA-256 digest, and byte size of a file under cache/.
type FileChecksum struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

// Validate checks required manifest fields and artifact invariants.
func (m *Manifest) Validate() error {
	if m == nil {
		return fmt.Errorf("%w: nil manifest", ErrInvalidManifest)
	}
	if strings.TrimSpace(m.BundleID) == "" {
		return fmt.Errorf("%w: missing bundle_id", ErrInvalidManifest)
	}
	if strings.TrimSpace(m.SiteID) == "" {
		return fmt.Errorf("%w: missing site_id", ErrInvalidManifest)
	}
	if m.Sequence == 0 {
		return fmt.Errorf("%w: sequence must be greater than zero", ErrInvalidManifest)
	}
	if strings.TrimSpace(m.TargetSystem) == "" {
		return fmt.Errorf("%w: missing target_system", ErrInvalidManifest)
	}
	if err := validateModels(m.Models); err != nil {
		return err
	}
	return validateCacheFiles(m.CacheFiles)
}

func validateModels(models []ModelArtifact) error {
	for i, model := range models {
		if !IsHexSHA256(model.SHA256) {
			return fmt.Errorf("%w: models[%d] has invalid sha256 %q", ErrInvalidManifest, i, model.SHA256)
		}
		if model.Size < 0 {
			return fmt.Errorf("%w: models[%d] has negative size %d", ErrInvalidManifest, i, model.Size)
		}
	}
	return nil
}

func validateCacheFiles(files []FileChecksum) error {
	for i, cf := range files {
		if !fs.ValidPath(cf.Path) || cf.Path == "." {
			return fmt.Errorf("%w: cache_files[%d] has invalid path %q", ErrInvalidManifest, i, cf.Path)
		}
		if !IsHexSHA256(cf.SHA256) {
			return fmt.Errorf("%w: cache_files[%d] has invalid sha256 %q", ErrInvalidManifest, i, cf.SHA256)
		}
		if cf.Size < 0 {
			return fmt.Errorf("%w: cache_files[%d] has negative size %d", ErrInvalidManifest, i, cf.Size)
		}
	}
	return nil
}

// IsHexSHA256 reports whether s is a 64-character lowercase hex SHA-256 digest.
func IsHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := range len(s) {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// WriteManifest validates m and writes it deterministically and atomically to path.
func WriteManifest(path string, m *Manifest) error {
	if err := m.Validate(); err != nil {
		return err
	}

	normalized := *m
	if len(m.Models) > 0 {
		normalized.Models = slices.Clone(m.Models)
		slices.SortFunc(normalized.Models, func(a, b ModelArtifact) int {
			return strings.Compare(a.SHA256, b.SHA256)
		})
	}
	if len(m.CacheFiles) > 0 {
		normalized.CacheFiles = slices.Clone(m.CacheFiles)
		slices.SortFunc(normalized.CacheFiles, func(a, b FileChecksum) int {
			return strings.Compare(a.Path, b.Path)
		})
	}

	data, err := json.MarshalIndent(&normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}
	data = append(data, '\n')
	return writeFileAtomic(path, data, 0o644)
}

// ReadManifest reads, unmarshals, and validates a manifest file.
// Missing, empty, or truncated manifest files return an error wrapping ErrIncomplete.
func ReadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: missing %s", ErrIncomplete, ManifestFile)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, fmt.Errorf("%w: empty %s", ErrIncomplete, ManifestFile)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		if strings.Contains(err.Error(), "unexpected end of JSON input") {
			return nil, fmt.Errorf("%w: truncated %s: %w", ErrIncomplete, ManifestFile, err)
		}
		return nil, fmt.Errorf("%w: parse %s: %w", ErrCorrupt, ManifestFile, err)
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// Verify checks that bundleDir contains a valid manifest matching expectedSiteID
// (when non-empty) and that every listed model and cache file is present with matching size and SHA-256.
func Verify(bundleDir, expectedSiteID string) (*Manifest, error) {
	m, err := ReadManifest(filepath.Join(bundleDir, ManifestFile))
	if err != nil {
		return nil, err
	}
	if expectedSiteID != "" && m.SiteID != expectedSiteID {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrSiteMismatch, m.SiteID, expectedSiteID)
	}

	for _, model := range m.Models {
		modelPath := filepath.Join(bundleDir, ModelsDir, model.SHA256)
		if err := verifyFile(modelPath, model.SHA256, model.Size); err != nil {
			return nil, fmt.Errorf("model %s: %w", model.SHA256, err)
		}
	}

	for _, cf := range m.CacheFiles {
		cachePath := filepath.Join(bundleDir, CacheDir, filepath.FromSlash(cf.Path))
		if err := verifyFile(cachePath, cf.SHA256, cf.Size); err != nil {
			return nil, fmt.Errorf("cache file %s: %w", cf.Path, err)
		}
	}

	return m, nil
}

func verifyFile(path, expectedSHA256 string, expectedSize int64) error {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: missing file", ErrIncomplete)
		}
		return fmt.Errorf("stat file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: not a regular file", ErrCorrupt)
	}
	if info.Size() < expectedSize {
		return fmt.Errorf("%w: got %d bytes, expected %d", ErrIncomplete, info.Size(), expectedSize)
	}
	if info.Size() > expectedSize {
		return fmt.Errorf("%w: size %d exceeds expected %d", ErrCorrupt, info.Size(), expectedSize)
	}

	sum, _, err := HashFile(path)
	if err != nil {
		return err
	}
	if sum != expectedSHA256 {
		return fmt.Errorf("%w: sha256 %s != expected %s", ErrCorrupt, sum, expectedSHA256)
	}
	return nil
}

// HashFile computes the lowercase hex SHA-256 digest and byte size of path by streaming.
func HashFile(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

// ScanCacheDir hashes all regular files under cacheDir and returns them sorted by relative path.
func ScanCacheDir(cacheDir string) ([]FileChecksum, error) {
	var files []FileChecksum
	err := filepath.WalkDir(cacheDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("unsupported non-regular cache entry %q", path)
		}
		rel, err := filepath.Rel(cacheDir, path)
		if err != nil {
			return err
		}
		sum, size, err := HashFile(path)
		if err != nil {
			return err
		}
		files = append(files, FileChecksum{
			Path:   filepath.ToSlash(rel),
			SHA256: sum,
			Size:   size,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan cache dir: %w", err)
	}
	slices.SortFunc(files, func(a, b FileChecksum) int {
		return strings.Compare(a.Path, b.Path)
	})
	return files, nil
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".tmp-*")
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
