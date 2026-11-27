package status

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStatusDirRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "status-out")
	want := &Bundle{
		SiteID:              "site-a",
		LastAppliedSequence: 2,
		CurrentSystem:       "/nix/store/bbbb-nixos-system-site-a",
		CurrentGeneration:   3,
		CurrentModelSHA256:  "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db",
		Health: Health{
			Healthy: true,
			Message: "ok",
		},
		RolloutEvents: []RolloutEvent{
			{
				Sequence:     2,
				BundleID:     "site-a-0002",
				TargetSystem: "/nix/store/bbbb-nixos-system-site-a",
				ModelSHA256:  "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db",
				Outcome:      "applied",
			},
		},
		AgentLogTail: []string{
			"verified bundle site-a-0002",
			"activated generation 3",
		},
	}

	if err := WriteDir(dir, want); err != nil {
		t.Fatalf("WriteDir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != StatusFile {
		t.Fatalf("expected only %s in status dir, got %v", StatusFile, entries)
	}

	got, err := ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("status mismatch:\ngot:  %+v\nwant: %+v", got, want)
	}
}

func TestStatusValidationErrors(t *testing.T) {
	dir := t.TempDir()

	if err := WriteDir(dir, &Bundle{}); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus for empty site_id, got %v", err)
	}

	bad := &Bundle{
		SiteID:            "site-a",
		CurrentGeneration: -1,
	}
	if err := WriteDir(dir, bad); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus for negative generation, got %v", err)
	}

	badEvent := &Bundle{
		SiteID: "site-a",
		RolloutEvents: []RolloutEvent{
			{Sequence: 1},
		},
	}
	if err := WriteDir(dir, badEvent); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus for event without outcome, got %v", err)
	}

	badJSONPath := filepath.Join(dir, StatusFile)
	if err := os.WriteFile(badJSONPath, []byte("{bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadDir(dir); !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("expected ErrInvalidStatus for corrupt json, got %v", err)
	}
}
