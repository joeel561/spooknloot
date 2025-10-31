package assetpack

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// Embedded holds all game assets and required data files.
// Patterns are relative to this file's directory (project root).
//
//go:embed assets/** pkg/world/map.json pkg/boss/map.json spooknloot.json
var Embedded embed.FS

// Prepare extracts embedded assets into a temporary directory and switches
// the current working directory there so the rest of the game can keep using
// existing relative file paths (e.g., assets/..., pkg/**/map.json).
// It returns the extraction directory, a cleanup function, and an error.
func Prepare() (string, func(), error) {
	// Create temp dir
	tmpDir, err := os.MkdirTemp("", "spooknloot_assets_*")
	if err != nil {
		return "", nil, err
	}

	// Walk embedded FS and write files
	if err := fs.WalkDir(Embedded, ".", func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return os.MkdirAll(filepath.Join(tmpDir, p), 0o755)
		}
		// Copy file
		src, err := Embedded.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		dstPath := filepath.Join(tmpDir, p)
		if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
			return err
		}
		dst, err := os.Create(dstPath)
		if err != nil {
			return err
		}
		if _, err := io.Copy(dst, src); err != nil {
			dst.Close()
			return err
		}
		return dst.Close()
	}); err != nil {
		return "", nil, err
	}

	// Switch working directory so all relative loads keep working
	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		return "", nil, err
	}

	cleanup := func() {
		// Try to return to previous working directory, then remove temp dir
		_ = os.Chdir(oldWD)
		// See note in previous build: we avoid removing to prevent issues with
		// backend file handles; uncomment to clean up automatically.
		// _ = os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup, nil
}
