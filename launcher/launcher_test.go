package launcher

import (
	"testing"
)

func TestValidateRepo(t *testing.T) {
	tests := []struct {
		name    string
		repo    string
		wantErr bool
	}{
		{"valid repo", "owner/repo", false},
		{"valid with dashes", "my-org/my-repo", false},
		{"empty", "", true},
		{"no slash", "ownerrepo", true},
		{"slash at start", "/repo", true},
		{"slash at end", "owner/", true},
		{"just slash", "/", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRepo(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRepo(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "fix-bug", false},
		{"valid with numbers", "feature123", false},
		{"empty", "", true},
		{"with space", "fix bug", true},
		{"with slash", "fix/bug", true},
		{"with backslash", "fix\\bug", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePrompt(t *testing.T) {
	tests := []struct {
		name    string
		prompt  string
		wantErr bool
	}{
		{"valid prompt", "Fix the login bug", false},
		{"long prompt", "This is a very long prompt that describes what the AI should do in detail", false},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePrompt(tt.prompt)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePrompt(%q) error = %v, wantErr %v", tt.prompt, err, tt.wantErr)
			}
		})
	}
}

func TestLaunch_ValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		opts Options
	}{
		{
			name: "empty repo",
			opts: Options{
				Repo:   "",
				Name:   "test",
				Prompt: "test prompt",
			},
		},
		{
			name: "invalid repo",
			opts: Options{
				Repo:   "invalidrepo",
				Name:   "test",
				Prompt: "test prompt",
			},
		},
		{
			name: "empty name",
			opts: Options{
				Repo:   "owner/repo",
				Name:   "",
				Prompt: "test prompt",
			},
		},
		{
			name: "invalid name",
			opts: Options{
				Repo:   "owner/repo",
				Name:   "test name",
				Prompt: "test prompt",
			},
		},
		{
			name: "empty prompt",
			opts: Options{
				Repo:   "owner/repo",
				Name:   "test",
				Prompt: "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Launch(tt.opts)
			if result.Error == nil {
				t.Error("expected validation error")
			}
		})
	}
}

func TestOptions_Fields(t *testing.T) {
	opts := Options{
		Repo:       "owner/repo",
		Name:       "test",
		Prompt:     "test prompt",
		PresetName: "default",
		Multiplier: 2,
		DataDir:    "/tmp/test",
	}

	if opts.Repo != "owner/repo" {
		t.Error("Repo field not set correctly")
	}
	if opts.Name != "test" {
		t.Error("Name field not set correctly")
	}
	if opts.Prompt != "test prompt" {
		t.Error("Prompt field not set correctly")
	}
	if opts.PresetName != "default" {
		t.Error("PresetName field not set correctly")
	}
	if opts.Multiplier != 2 {
		t.Error("Multiplier field not set correctly")
	}
	if opts.DataDir != "/tmp/test" {
		t.Error("DataDir field not set correctly")
	}
}

func TestResult_Fields(t *testing.T) {
	result := Result{
		RepoPath:            "/tmp/repo",
		TerminalWindowCount: 2,
	}

	if result.RepoPath != "/tmp/repo" {
		t.Error("RepoPath field not set correctly")
	}
	if result.TerminalWindowCount != 2 {
		t.Error("TerminalWindowCount field not set correctly")
	}
	if result.Error != nil {
		t.Error("Error should be nil")
	}
}
