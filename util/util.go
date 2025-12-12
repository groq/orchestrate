// Package util provides utility functions for orchestrate.
package util

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

// RandomReader is used to generate random bytes (can be replaced for testing).
var RandomReader io.Reader = rand.Reader

// RandomHex generates a random hex string of n bytes (2n hex characters).
func RandomHex(n int) string {
	bytes := make([]byte, n)
	_, _ = RandomReader.Read(bytes)
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
