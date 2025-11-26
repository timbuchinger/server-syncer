package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"agent-align/internal/config"
)

func copyExtraFileTarget(target config.ExtraFileTarget) error {
	info, err := os.Stat(target.Source)
	if err != nil {
		return fmt.Errorf("failed to inspect %s: %w", target.Source, err)
	}
	if info.IsDir() {
		return fmt.Errorf("extra file target %s is a directory; use directories instead", target.Source)
	}
	for _, dest := range target.Destinations {
		if err := copyFileContents(target.Source, dest, info.Mode()); err != nil {
			return fmt.Errorf("failed to copy %s to %s: %w", target.Source, dest, err)
		}
	}
	return nil
}

func copyExtraDirectoryTarget(target config.ExtraDirectoryTarget) (int, error) {
	sourceInfo, err := os.Stat(target.Source)
	if err != nil {
		return 0, fmt.Errorf("failed to inspect %s: %w", target.Source, err)
	}
	if !sourceInfo.IsDir() {
		return 0, fmt.Errorf("extra directory target %s is not a directory", target.Source)
	}

	var total int
	for _, dest := range target.Destinations {
		count, err := copyDirectory(target.Source, dest.Path, dest.Flatten)
		if err != nil {
			return total, fmt.Errorf("failed to copy directory %s to %s: %w", target.Source, dest.Path, err)
		}
		total += count
	}
	return total, nil
}

func copyDirectory(source, destination string, flatten bool) (int, error) {
	var copied int
	walkErr := filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		var destPath string
		if flatten {
			destPath = filepath.Join(destination, filepath.Base(path))
		} else {
			rel, err := filepath.Rel(source, path)
			if err != nil {
				return err
			}
			destPath = filepath.Join(destination, rel)
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := copyFileContents(path, destPath, info.Mode()); err != nil {
			return err
		}
		copied++
		return nil
	})
	if walkErr != nil {
		return copied, walkErr
	}
	return copied, nil
}

func copyFileContents(source, dest string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("failed to create directory for %s: %w", dest, err)
	}

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("failed to copy %s to %s: %w", source, dest, err)
	}
	return nil
}
