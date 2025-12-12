package util

import (
	"bytes"
	"io"
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
