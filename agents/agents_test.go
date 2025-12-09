package agents

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single agent",
			input: "claude",
			want:  []string{"claude"},
		},
		{
			name:  "multiple agents",
			input: "claude,codex,droid",
			want:  []string{"claude", "codex", "droid"},
		},
		{
			name:  "with spaces",
			input: "claude, codex, droid",
			want:  []string{"claude", "codex", "droid"},
		},
		{
			name:  "with brackets",
			input: "[claude,codex]",
			want:  []string{"claude", "codex"},
		},
		{
			name:  "with brackets and spaces",
			input: "[ claude, codex, droid ]",
			want:  []string{"claude", "codex", "droid"},
		},
		{
			name:  "duplicate agents",
			input: "claude,claude,claude",
			want:  []string{"claude", "claude", "claude"},
		},
		{
			name:  "empty string",
			input: "",
			want:  nil,
		},
		{
			name:  "only commas",
			input: ",,,",
			want:  nil,
		},
		{
			name:  "trailing comma",
			input: "claude,codex,",
			want:  []string{"claude", "codex"},
		},
		{
			name:  "leading comma",
			input: ",claude,codex",
			want:  []string{"claude", "codex"},
		},
		{
			name:  "extra whitespace",
			input: "  claude  ,  codex  ",
			want:  []string{"claude", "codex"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Parse(tt.input)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetColor(t *testing.T) {
	tests := []struct {
		name   string
		agent  string
		wantR  int
		wantG  int
		wantB  int
		wantOk bool
	}{
		{
			name:   "droid - orange",
			agent:  "droid",
			wantR:  255,
			wantG:  140,
			wantB:  0,
			wantOk: true,
		},
		{
			name:   "claude - sand/tan",
			agent:  "claude",
			wantR:  210,
			wantG:  180,
			wantB:  140,
			wantOk: true,
		},
		{
			name:   "codex - black",
			agent:  "codex",
			wantR:  30,
			wantG:  30,
			wantB:  30,
			wantOk: true,
		},
		{
			name:   "unknown agent",
			agent:  "unknown",
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color, ok := GetColor(tt.agent)
			if ok != tt.wantOk {
				t.Errorf("GetColor(%q) ok = %v, want %v", tt.agent, ok, tt.wantOk)
			}
			if tt.wantOk {
				if color.R != tt.wantR || color.G != tt.wantG || color.B != tt.wantB {
					t.Errorf("GetColor(%q) = RGB(%d,%d,%d), want RGB(%d,%d,%d)",
						tt.agent, color.R, color.G, color.B, tt.wantR, tt.wantG, tt.wantB)
				}
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name   string
		agents []string
		want   string
	}{
		{
			name:   "single agent",
			agents: []string{"claude"},
			want:   "claude",
		},
		{
			name:   "multiple agents",
			agents: []string{"claude", "codex", "droid"},
			want:   "claude, codex, droid",
		},
		{
			name:   "empty slice",
			agents: []string{},
			want:   "",
		},
		{
			name:   "nil slice",
			agents: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Format(tt.agents)
			if got != tt.want {
				t.Errorf("Format(%v) = %q, want %q", tt.agents, got, tt.want)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	tests := []struct {
		name   string
		agents []string
		n      int
		want   []string
	}{
		{
			name:   "expand by 1",
			agents: []string{"claude", "codex"},
			n:      1,
			want:   []string{"claude", "codex"},
		},
		{
			name:   "expand by 2",
			agents: []string{"claude", "codex"},
			n:      2,
			want:   []string{"claude", "codex", "claude", "codex"},
		},
		{
			name:   "expand by 3",
			agents: []string{"droid"},
			n:      3,
			want:   []string{"droid", "droid", "droid"},
		},
		{
			name:   "expand by 0 (treated as 1)",
			agents: []string{"claude"},
			n:      0,
			want:   []string{"claude"},
		},
		{
			name:   "expand by negative (treated as 1)",
			agents: []string{"claude"},
			n:      -1,
			want:   []string{"claude"},
		},
		{
			name:   "empty agents",
			agents: []string{},
			n:      3,
			want:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Expand(tt.agents, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expand(%v, %d) = %v, want %v", tt.agents, tt.n, got, tt.want)
			}
		})
	}
}

func TestDefaultColors(t *testing.T) {
	// Verify all expected agents have colors
	expectedAgents := []string{"droid", "claude", "codex"}

	for _, agent := range expectedAgents {
		if _, ok := DefaultColors[agent]; !ok {
			t.Errorf("DefaultColors missing agent %q", agent)
		}
	}

	// Verify the exact number of colors
	if len(DefaultColors) != len(expectedAgents) {
		t.Errorf("DefaultColors has %d entries, expected %d", len(DefaultColors), len(expectedAgents))
	}
}

func TestRGBColor_Fields(t *testing.T) {
	color := RGBColor{R: 100, G: 150, B: 200}

	if color.R != 100 {
		t.Errorf("RGBColor.R = %d, want 100", color.R)
	}
	if color.G != 150 {
		t.Errorf("RGBColor.G = %d, want 150", color.G)
	}
	if color.B != 200 {
		t.Errorf("RGBColor.B = %d, want 200", color.B)
	}
}
