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
// - macOS: ~/Library/Application Support/Orchestrate
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
		baseDir = filepath.Join(home, "Library", "Application Support", "Orchestrate")
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
