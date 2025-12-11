// Package util provides utility functions for orchestrate.
package util

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// RandomReader is used to generate random bytes (can be replaced for testing).
var RandomReader io.Reader = rand.Reader

// RandomHex generates a random hex string of n bytes (2n hex characters).
func RandomHex(n int) string {
	bytes := make([]byte, n)
	RandomReader.Read(bytes)
	return hex.EncodeToString(bytes)
}

// SetRandomReader sets a custom random reader (useful for testing).
func SetRandomReader(r io.Reader) {
	RandomReader = r
}

// ResetRandomReader resets to the default crypto/rand reader.
func ResetRandomReader() {
	RandomReader = rand.Reader
}

// DataDir returns the platform-appropriate directory for orchestrate data.
// - macOS: ~/.orchestrate (avoids spaces in path for compatibility)
// - Linux: ~/.local/share/orchestrate (or $XDG_DATA_HOME/orchestrate)
// - Windows: %APPDATA%\Orchestrate
func DataDir() (string, error) {
	var baseDir string

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		baseDir = filepath.Join(home, ".orchestrate")
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		baseDir = filepath.Join(appData, "Orchestrate")
	default: // Linux and others
		xdgData := os.Getenv("XDG_DATA_HOME")
		if xdgData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			xdgData = filepath.Join(home, ".local", "share")
		}
		baseDir = filepath.Join(xdgData, "orchestrate")
	}

	return baseDir, nil
}

// DisplayPath converts an absolute path to a user-friendly display format.
// On Unix-like systems (macOS, Linux), it replaces the home directory with ~.
// On Windows, it returns the path unchanged.
func DisplayPath(path string) string {
	return DisplayPathWithHome(path, "")
}

// DisplayPathWithHome converts an absolute path to a user-friendly display format,
// using the provided home directory. If homeDir is empty, it uses os.UserHomeDir().
// This variant is useful for testing with different home directories.
func DisplayPathWithHome(path, homeDir string) string {
	if homeDir == "" {
		var err error
		homeDir, err = os.UserHomeDir()
		if err != nil {
			return path
		}
	}

	// On Windows, don't replace with ~ (not a common convention)
	if runtime.GOOS == "windows" {
		return path
	}

	// Try to make the path relative to home
	rel, err := filepath.Rel(homeDir, path)
	if err != nil {
		return path
	}

	// If the relative path starts with "..", it's not under home
	if len(rel) >= 2 && rel[0:2] == ".." {
		return path
	}

	return "~/" + rel
}
