package main

import (
	"os"
	"path/filepath"
	"testing"

	"agent-align/internal/config"
)

func TestCopyExtraFileTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "AGENTS.md")
	if err := os.WriteFile(source, []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	dest1 := filepath.Join(dir, "dest", "one.md")
	dest2 := filepath.Join(dir, "dest-two.md")
	target := config.ExtraFileTarget{
		Source:       source,
		Destinations: []string{dest1, dest2},
	}
	if err := copyExtraFileTarget(target); err != nil {
		t.Fatalf("copyExtraFileTarget returned error: %v", err)
	}

	for _, dest := range []string{dest1, dest2} {
		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("failed to read %s: %v", dest, err)
		}
		if string(data) != "hello" {
			t.Fatalf("unexpected file contents for %s: %q", dest, data)
		}
	}
}

func TestCopyExtraDirectoryTarget(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "root.txt"), []byte("root"), 0o644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	dest := filepath.Join(dir, "dest")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 files copied, got %d", count)
	}

	wantFiles := []string{
		filepath.Join(dest, "root.txt"),
		filepath.Join(dest, "nested", "child.txt"),
	}
	for _, path := range wantFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestCopyExtraDirectoryTargetMultipleDestinations(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatalf("failed to create source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "one.txt"), []byte("one"), 0o644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	dest1 := filepath.Join(dir, "dest1")
	dest2 := filepath.Join(dir, "dest2")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest1},
			{Path: dest2},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected total copies to be 2, got %d", count)
	}

	for _, dest := range []string{dest1, dest2} {
		if _, err := os.Stat(filepath.Join(dest, "one.txt")); err != nil {
			t.Fatalf("expected %s to contain copied file: %v", dest, err)
		}
	}
}

func TestCopyExtraDirectoryTargetFlatten(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "src")
	if err := os.MkdirAll(filepath.Join(source, "nested"), 0o755); err != nil {
		t.Fatalf("failed to create source tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(source, "nested", "child.txt"), []byte("child"), 0o644); err != nil {
		t.Fatalf("failed to write nested file: %v", err)
	}

	dest := filepath.Join(dir, "dest")
	target := config.ExtraDirectoryTarget{
		Source: source,
		Destinations: []config.ExtraDirectoryCopyRoute{
			{Path: dest, Flatten: true},
		},
	}
	count, err := copyExtraDirectoryTarget(target)
	if err != nil {
		t.Fatalf("copyExtraDirectoryTarget returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 file copied, got %d", count)
	}

	if _, err := os.Stat(filepath.Join(dest, "child.txt")); err != nil {
		t.Fatalf("expected flattened file to exist: %v", err)
	}
}
