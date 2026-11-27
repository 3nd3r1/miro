package bundle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func sha256Bytes(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func createValidBundle(t *testing.T) (string, *Manifest) {
	t.Helper()
	dir := t.TempDir()

	modelBytes := []byte("fake-gguf-model-weights-v1")
	modelSum := sha256Bytes(modelBytes)
	modelPath := filepath.Join(dir, ModelsDir, modelSum)
	if err := os.MkdirAll(filepath.Dir(modelPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(modelPath, modelBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	cacheInfo := []byte("StoreDir: /nix/store\n")
	narBytes := []byte("nar-payload-bytes")
	cacheDir := filepath.Join(dir, CacheDir)
	if err := os.MkdirAll(filepath.Join(cacheDir, "nar"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "nix-cache-info"), cacheInfo, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "nar", "abc.nar"), narBytes, 0o644); err != nil {
		t.Fatal(err)
	}

	cacheFiles, err := ScanCacheDir(cacheDir)
	if err != nil {
		t.Fatalf("ScanCacheDir: %v", err)
	}

	m := &Manifest{
		BundleID:     "site-a-0001",
		SiteID:       "site-a",
		Sequence:     1,
		TargetSystem: "/nix/store/aaaa-nixos-system-site-a",
		BaseSystem:   "/nix/store/0000-nixos-system-site-a",
		Models: []ModelArtifact{
			{
				Name:   "tiny.gguf",
				SHA256: modelSum,
				Size:   int64(len(modelBytes)),
			},
		},
		CacheFiles: cacheFiles,
		CreatedAt:  "2026-11-27T12:00:00Z",
	}
	if err := WriteManifest(filepath.Join(dir, ManifestFile), m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
	return dir, m
}

func TestManifestRoundTripAndVerify(t *testing.T) {
	dir, want := createValidBundle(t)

	got, err := Verify(dir, "site-a")
	if err != nil {
		t.Fatalf("Verify returned unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestWriteManifestDeterministicOrder(t *testing.T) {
	dir := t.TempDir()
	path1 := filepath.Join(dir, "m1.json")
	path2 := filepath.Join(dir, "m2.json")

	sumA := sha256Bytes([]byte("a"))
	sumB := sha256Bytes([]byte("b"))

	m1 := &Manifest{
		BundleID:     "b1",
		SiteID:       "site-a",
		Sequence:     1,
		TargetSystem: "/nix/store/sys",
		Models: []ModelArtifact{
			{SHA256: sumB, Size: 1},
			{SHA256: sumA, Size: 1},
		},
		CacheFiles: []FileChecksum{
			{Path: "z.narinfo", SHA256: sumB, Size: 1},
			{Path: "a.narinfo", SHA256: sumA, Size: 1},
		},
	}
	m2 := &Manifest{
		BundleID:     "b1",
		SiteID:       "site-a",
		Sequence:     1,
		TargetSystem: "/nix/store/sys",
		Models: []ModelArtifact{
			{SHA256: sumA, Size: 1},
			{SHA256: sumB, Size: 1},
		},
		CacheFiles: []FileChecksum{
			{Path: "a.narinfo", SHA256: sumA, Size: 1},
			{Path: "z.narinfo", SHA256: sumB, Size: 1},
		},
	}

	if err := WriteManifest(path1, m1); err != nil {
		t.Fatal(err)
	}
	if err := WriteManifest(path2, m2); err != nil {
		t.Fatal(err)
	}

	b1, err := os.ReadFile(path1)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := os.ReadFile(path2)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("manifest JSON not deterministic:\n%s\nvs\n%s", b1, b2)
	}
}

func TestVerifyIncompleteCases(t *testing.T) {
	t.Run("missing manifest", func(t *testing.T) {
		dir := t.TempDir()
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})

	t.Run("truncated manifest", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`{"bundle_id": "x"`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})

	t.Run("missing model file", func(t *testing.T) {
		dir, m := createValidBundle(t)
		if err := os.Remove(filepath.Join(dir, ModelsDir, m.Models[0].SHA256)); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})

	t.Run("truncated model file", func(t *testing.T) {
		dir, m := createValidBundle(t)
		modelPath := filepath.Join(dir, ModelsDir, m.Models[0].SHA256)
		if err := os.WriteFile(modelPath, []byte("short"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})

	t.Run("missing cache file", func(t *testing.T) {
		dir, _ := createValidBundle(t)
		if err := os.Remove(filepath.Join(dir, CacheDir, "nar", "abc.nar")); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})

	t.Run("truncated cache file", func(t *testing.T) {
		dir, _ := createValidBundle(t)
		if err := os.WriteFile(filepath.Join(dir, CacheDir, "nar", "abc.nar"), []byte("nar"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrIncomplete) {
			t.Fatalf("expected ErrIncomplete, got %v", err)
		}
	})
}

func TestVerifyCorruptAndMismatchCases(t *testing.T) {
	t.Run("site id mismatch", func(t *testing.T) {
		dir, _ := createValidBundle(t)
		_, err := Verify(dir, "site-b")
		if !errors.Is(err, ErrSiteMismatch) {
			t.Fatalf("expected ErrSiteMismatch, got %v", err)
		}
	})

	t.Run("model checksum mismatch same length", func(t *testing.T) {
		dir, m := createValidBundle(t)
		modelPath := filepath.Join(dir, ModelsDir, m.Models[0].SHA256)
		bad := []byte("Xake-gguf-model-weights-v1")
		if int64(len(bad)) != m.Models[0].Size {
			t.Fatalf("test setup error: length mismatch")
		}
		if err := os.WriteFile(modelPath, bad, 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt, got %v", err)
		}
	})

	t.Run("cache checksum mismatch", func(t *testing.T) {
		dir, _ := createValidBundle(t)
		cachePath := filepath.Join(dir, CacheDir, "nar", "abc.nar")
		if err := os.WriteFile(cachePath, []byte("Xar-payload-bytes"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt, got %v", err)
		}
	})

	t.Run("oversized cache file", func(t *testing.T) {
		dir, _ := createValidBundle(t)
		cachePath := filepath.Join(dir, CacheDir, "nar", "abc.nar")
		if err := os.WriteFile(cachePath, []byte("nar-payload-bytes-extra"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt, got %v", err)
		}
	})

	t.Run("malformed json manifest", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(`not-json`), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrCorrupt) {
			t.Fatalf("expected ErrCorrupt, got %v", err)
		}
	})

	t.Run("path traversal in cache file", func(t *testing.T) {
		dir := t.TempDir()
		badManifest := `{"bundle_id":"b1","site_id":"site-a","sequence":1,"target_system":"/nix/store/x","cache_files":[{"path":"../escape","sha256":"` + sha256Bytes([]byte("x")) + `","size":1}]}`
		if err := os.WriteFile(filepath.Join(dir, ManifestFile), []byte(badManifest), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Verify(dir, "site-a")
		if !errors.Is(err, ErrInvalidManifest) {
			t.Fatalf("expected ErrInvalidManifest, got %v", err)
		}
	})
}
