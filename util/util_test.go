package util

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"testing"
)

// mockReader is a simple io.Reader that returns predetermined bytes
type mockReader struct {
	data []byte
	pos  int
}

func (m *mockReader) Read(p []byte) (n int, err error) {
	if m.pos >= len(m.data) {
		return 0, io.EOF
	}
	n = copy(p, m.data[m.pos:])
	m.pos += n
	return n, nil
}

func TestRandomHex(t *testing.T) {
	tests := []struct {
		name       string
		n          int
		mockData   []byte
		wantLength int
		wantHex    string
	}{
		{
			name:       "4 bytes = 8 hex chars",
			n:          4,
			mockData:   []byte{0xab, 0xcd, 0xef, 0x12},
			wantLength: 8,
			wantHex:    "abcdef12",
		},
		{
			name:       "2 bytes = 4 hex chars",
			n:          2,
			mockData:   []byte{0x00, 0xff},
			wantLength: 4,
			wantHex:    "00ff",
		},
		{
			name:       "1 byte = 2 hex chars",
			n:          1,
			mockData:   []byte{0x5a},
			wantLength: 2,
			wantHex:    "5a",
		},
		{
			name:       "8 bytes = 16 hex chars",
			n:          8,
			mockData:   []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef},
			wantLength: 16,
			wantHex:    "0123456789abcdef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up mock reader
			SetRandomReader(&mockReader{data: tt.mockData})
			defer ResetRandomReader()

			got := RandomHex(tt.n)
			if len(got) != tt.wantLength {
				t.Errorf("RandomHex(%d) length = %d, want %d", tt.n, len(got), tt.wantLength)
			}
			if got != tt.wantHex {
				t.Errorf("RandomHex(%d) = %q, want %q", tt.n, got, tt.wantHex)
			}
		})
	}
}

func TestRandomHex_RealRandom(t *testing.T) {
	// Test with real random reader
	ResetRandomReader()

	// Generate multiple random hex strings and verify they're unique
	results := make(map[string]bool)
	for i := 0; i < 100; i++ {
		hex := RandomHex(4)
		if len(hex) != 8 {
			t.Errorf("RandomHex(4) length = %d, want 8", len(hex))
		}
		// Check it's valid hex
		for _, c := range hex {
			if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
				t.Errorf("RandomHex(4) = %q contains invalid hex char %c", hex, c)
			}
		}
		results[hex] = true
	}

	// With 8 hex chars (32 bits of randomness), collisions should be extremely rare
	if len(results) < 95 {
		t.Errorf("RandomHex generated too many duplicates: %d unique out of 100", len(results))
	}
}

func TestRandomHex_ZeroBytes(t *testing.T) {
	SetRandomReader(&mockReader{data: []byte{}})
	defer ResetRandomReader()

	got := RandomHex(0)
	if got != "" {
		t.Errorf("RandomHex(0) = %q, want empty string", got)
	}
}

func TestSetAndResetRandomReader(t *testing.T) {
	// Create a predictable reader
	predictable := bytes.NewReader([]byte{0xde, 0xad, 0xbe, 0xef})
	SetRandomReader(predictable)

	hex := RandomHex(4)
	if hex != "deadbeef" {
		t.Errorf("With mock reader, RandomHex(4) = %q, want 'deadbeef'", hex)
	}

	// Reset and verify it still works (generates random data)
	ResetRandomReader()
	hex = RandomHex(4)
	if len(hex) != 8 {
		t.Errorf("After reset, RandomHex(4) length = %d, want 8", len(hex))
	}
}

func TestRandomHex_ValidHexOutput(t *testing.T) {
	ResetRandomReader()

	// Test various sizes
	sizes := []int{1, 2, 4, 8, 16, 32}
	validHex := "0123456789abcdef"

	for _, size := range sizes {
		hex := RandomHex(size)

		// Check length
		if len(hex) != size*2 {
			t.Errorf("RandomHex(%d) length = %d, want %d", size, len(hex), size*2)
		}

		// Check all characters are valid hex
		for i, c := range hex {
			found := false
			for _, valid := range validHex {
				if c == valid {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("RandomHex(%d)[%d] = %c, not a valid hex char", size, i, c)
			}
		}
	}
}

func TestDataDir(t *testing.T) {
	dir, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir() error = %v", err)
	}

	if dir == "" {
		t.Error("DataDir() returned empty string")
	}

	// Verify platform-specific expectations
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(dir, ".orchestrate") {
			t.Errorf("DataDir() on macOS = %q, want to contain '.orchestrate'", dir)
		}
	case "windows":
		if !strings.Contains(dir, "Orchestrate") {
			t.Errorf("DataDir() on Windows = %q, want to contain 'Orchestrate'", dir)
		}
	default: // Linux
		if !strings.Contains(dir, "orchestrate") {
			t.Errorf("DataDir() on Linux = %q, want to contain 'orchestrate'", dir)
		}
	}
}

func TestDisplayPathWithHome(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		homeDir  string
		want     string
		skipOnOS string // skip this test on specific OS (e.g., "windows")
	}{
		{
			name:     "path under home directory - macOS style",
			path:     "/Users/testuser/.orchestrate/settings.yaml",
			homeDir:  "/Users/testuser",
			want:     "~/.orchestrate/settings.yaml",
			skipOnOS: "windows",
		},
		{
			name:     "path under home directory - Linux style",
			path:     "/home/testuser/.local/share/orchestrate/settings.yaml",
			homeDir:  "/home/testuser",
			want:     "~/.local/share/orchestrate/settings.yaml",
			skipOnOS: "windows",
		},
		{
			name:     "path exactly at home directory",
			path:     "/Users/testuser",
			homeDir:  "/Users/testuser",
			want:     "~/.",
			skipOnOS: "windows",
		},
		{
			name:     "path not under home directory",
			path:     "/var/log/something.log",
			homeDir:  "/Users/testuser",
			want:     "/var/log/something.log",
			skipOnOS: "windows",
		},
		{
			name:     "path is parent of home directory",
			path:     "/Users",
			homeDir:  "/Users/testuser",
			want:     "/Users",
			skipOnOS: "windows",
		},
		{
			name:     "nested path under home",
			path:     "/home/user/projects/myapp/src/main.go",
			homeDir:  "/home/user",
			want:     "~/projects/myapp/src/main.go",
			skipOnOS: "windows",
		},
		{
			name:     "home with trailing slash",
			path:     "/Users/testuser/.orchestrate/settings.yaml",
			homeDir:  "/Users/testuser/",
			want:     "~/.orchestrate/settings.yaml",
			skipOnOS: "windows",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnOS != "" && runtime.GOOS == tt.skipOnOS {
				t.Skipf("Skipping test on %s", runtime.GOOS)
			}

			got := DisplayPathWithHome(tt.path, tt.homeDir)
			if got != tt.want {
				t.Errorf("DisplayPathWithHome(%q, %q) = %q, want %q", tt.path, tt.homeDir, got, tt.want)
			}
		})
	}
}

func TestDisplayPathWithHome_Windows(t *testing.T) {
	// On Windows, DisplayPath should return the path unchanged (no ~ substitution)
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test on non-Windows OS")
	}

	tests := []struct {
		name    string
		path    string
		homeDir string
	}{
		{
			name:    "Windows path should not use tilde",
			path:    `C:\Users\testuser\AppData\Roaming\Orchestrate\settings.yaml`,
			homeDir: `C:\Users\testuser`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DisplayPathWithHome(tt.path, tt.homeDir)
			// On Windows, should return unchanged
			if got != tt.path {
				t.Errorf("DisplayPathWithHome on Windows should return unchanged path, got %q", got)
			}
		})
	}
}

func TestDisplayPath(t *testing.T) {
	// Test that DisplayPath uses the actual home directory
	got := DisplayPath("/some/random/path")

	// Should return the path (either with ~ if under home, or unchanged)
	if got == "" {
		t.Error("DisplayPath returned empty string")
	}

	// On non-Windows, if we pass a path under the actual home, it should start with ~/
	if runtime.GOOS != "windows" {
		dataDir, err := DataDir()
		if err == nil {
			displayed := DisplayPath(dataDir)
			if !strings.HasPrefix(displayed, "~/") {
				t.Errorf("DisplayPath(%q) = %q, expected to start with ~/", dataDir, displayed)
			}
		}
	}
}
