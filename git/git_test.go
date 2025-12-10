package git

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// MockCommander is a mock implementation of Commander for testing.
type MockCommander struct {
	RunFunc func(dir string, args ...string) (string, error)
	Calls   []MockCall
}

type MockCall struct {
	Dir  string
	Args []string
}

func (m *MockCommander) Run(dir string, args ...string) (string, error) {
	m.Calls = append(m.Calls, MockCall{Dir: dir, Args: args})
	if m.RunFunc != nil {
		return m.RunFunc(dir, args...)
	}
	return "", nil
}

func TestGetRootWithCmd(t *testing.T) {
	tests := []struct {
		name     string
		mockFunc func(dir string, args ...string) (string, error)
		want     string
		wantErr  bool
	}{
		{
			name: "successful get root",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "/path/to/repo", nil
			},
			want:    "/path/to/repo",
			wantErr: false,
		},
		{
			name: "not a git repo",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "", errors.New("not a git repo")
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "path with special chars",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "/path/to/my-repo_123", nil
			},
			want:    "/path/to/my-repo_123",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommander{RunFunc: tt.mockFunc}
			got, err := GetRootWithCmd(mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRootWithCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetRootWithCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetCurrentBranchWithCmd(t *testing.T) {
	tests := []struct {
		name     string
		mockFunc func(dir string, args ...string) (string, error)
		want     string
		wantErr  bool
	}{
		{
			name: "successful get branch",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "main", nil
			},
			want:    "main",
			wantErr: false,
		},
		{
			name: "feature branch",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "feature/new-feature", nil
			},
			want:    "feature/new-feature",
			wantErr: false,
		},
		{
			name: "error getting branch",
			mockFunc: func(dir string, args ...string) (string, error) {
				return "", errors.New("detached HEAD")
			},
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &MockCommander{RunFunc: tt.mockFunc}
			got, err := GetCurrentBranchWithCmd(mock)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCurrentBranchWithCmd() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetCurrentBranchWithCmd() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateWorktreeWithCmd(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("worktree path already exists", func(t *testing.T) {
		existingPath := filepath.Join(tmpDir, "existing")
		if err := os.MkdirAll(existingPath, 0755); err != nil {
			t.Fatal(err)
		}

		mock := &MockCommander{}
		err := CreateWorktreeWithCmd(mock, tmpDir, existingPath, "branch", "main")
		if err == nil {
			t.Error("Expected error when worktree path exists")
		}
		if len(mock.Calls) > 0 {
			t.Error("Should not have called git when path exists")
		}
	})

	t.Run("successful worktree creation", func(t *testing.T) {
		worktreePath := filepath.Join(tmpDir, "new-worktree")
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				return "Preparing worktree...", nil
			},
		}

		err := CreateWorktreeWithCmd(mock, tmpDir, worktreePath, "feature-branch", "main")
		if err != nil {
			t.Errorf("CreateWorktreeWithCmd() error = %v", err)
		}

		// Verify the call was made with correct args
		if len(mock.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
		}
		call := mock.Calls[0]
		if call.Dir != tmpDir {
			t.Errorf("Expected dir %s, got %s", tmpDir, call.Dir)
		}
		expectedArgs := []string{"worktree", "add", "-b", "feature-branch", worktreePath, "main"}
		if len(call.Args) != len(expectedArgs) {
			t.Errorf("Expected %d args, got %d", len(expectedArgs), len(call.Args))
		}
		for i, arg := range expectedArgs {
			if call.Args[i] != arg {
				t.Errorf("Arg %d: expected %s, got %s", i, arg, call.Args[i])
			}
		}
	})

	t.Run("git command fails", func(t *testing.T) {
		worktreePath := filepath.Join(tmpDir, "failed-worktree")
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				return "fatal: branch already exists", errors.New("exit status 128")
			},
		}

		err := CreateWorktreeWithCmd(mock, tmpDir, worktreePath, "existing-branch", "main")
		if err == nil {
			t.Error("Expected error when git command fails")
		}
	})
}

func TestWorktreeExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	existingPath := filepath.Join(tmpDir, "exists")
	if err := os.MkdirAll(existingPath, 0755); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "path exists",
			path: existingPath,
			want: true,
		},
		{
			name: "path does not exist",
			path: filepath.Join(tmpDir, "nonexistent"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreeExists(tt.path)
			if got != tt.want {
				t.Errorf("WorktreeExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetAndResetCommander(t *testing.T) {
	// Save original
	original := defaultCmd

	// Set custom commander
	mock := &MockCommander{}
	SetCommander(mock)

	if defaultCmd != mock {
		t.Error("SetCommander did not set the commander")
	}

	// Reset
	ResetCommander()

	if _, ok := defaultCmd.(*DefaultCommander); !ok {
		t.Error("ResetCommander did not reset to DefaultCommander")
	}

	// Restore original for other tests
	defaultCmd = original
}

func TestEnsureRepoWithCmd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("invalid repo format - no slash", func(t *testing.T) {
		mock := &MockCommander{}
		_, err := EnsureRepoWithCmd(mock, "invalidrepo", tmpDir)
		if err == nil {
			t.Error("Expected error for invalid repo format")
		}
	})

	t.Run("invalid repo format - too many parts", func(t *testing.T) {
		mock := &MockCommander{}
		_, err := EnsureRepoWithCmd(mock, "owner/repo/extra", tmpDir)
		if err == nil {
			t.Error("Expected error for invalid repo format")
		}
	})

	t.Run("successful clone of new repo", func(t *testing.T) {
		baseDir := filepath.Join(tmpDir, "repos1")
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				return "Cloning...", nil
			},
		}

		path, err := EnsureRepoWithCmd(mock, "groq/orion", baseDir)
		if err != nil {
			t.Errorf("EnsureRepoWithCmd() error = %v", err)
		}

		expectedPath := filepath.Join(baseDir, "groq-orion")
		if path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, path)
		}

		// Verify clone was called
		if len(mock.Calls) != 1 {
			t.Fatalf("Expected 1 call, got %d", len(mock.Calls))
		}
		call := mock.Calls[0]
		if call.Args[0] != "clone" {
			t.Errorf("Expected clone command, got %v", call.Args)
		}
	})

	t.Run("fetch and reset existing repo", func(t *testing.T) {
		baseDir := filepath.Join(tmpDir, "repos2")
		repoPath := filepath.Join(baseDir, "owner-repo")
		// Create existing repo directory with .git
		if err := os.MkdirAll(filepath.Join(repoPath, ".git"), 0755); err != nil {
			t.Fatal(err)
		}

		callCount := 0
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				callCount++
				return "OK", nil
			},
		}

		path, err := EnsureRepoWithCmd(mock, "owner/repo", baseDir)
		if err != nil {
			t.Errorf("EnsureRepoWithCmd() error = %v", err)
		}

		if path != repoPath {
			t.Errorf("Expected path %s, got %s", repoPath, path)
		}

		// Should have called fetch, reset, and clean
		if len(mock.Calls) != 3 {
			t.Errorf("Expected 3 calls (fetch, reset, clean), got %d", len(mock.Calls))
		}
	})

	t.Run("clone fails", func(t *testing.T) {
		baseDir := filepath.Join(tmpDir, "repos3")
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				return "fatal: repository not found", errors.New("exit status 128")
			},
		}

		_, err := EnsureRepoWithCmd(mock, "owner/nonexistent", baseDir)
		if err == nil {
			t.Error("Expected error when clone fails")
		}
	})
}

func TestFetchAndResetWithCmd(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "git_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("successful fetch and reset", func(t *testing.T) {
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				return "OK", nil
			},
		}

		err := FetchAndResetWithCmd(mock, tmpDir)
		if err != nil {
			t.Errorf("FetchAndResetWithCmd() error = %v", err)
		}

		// Verify commands were called in order
		if len(mock.Calls) != 3 {
			t.Fatalf("Expected 3 calls, got %d", len(mock.Calls))
		}

		// Check fetch
		if mock.Calls[0].Args[0] != "fetch" {
			t.Errorf("Expected fetch, got %v", mock.Calls[0].Args)
		}

		// Check reset
		if mock.Calls[1].Args[0] != "reset" {
			t.Errorf("Expected reset, got %v", mock.Calls[1].Args)
		}

		// Check clean
		if mock.Calls[2].Args[0] != "clean" {
			t.Errorf("Expected clean, got %v", mock.Calls[2].Args)
		}
	})

	t.Run("fetch fails", func(t *testing.T) {
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				if args[0] == "fetch" {
					return "fatal: could not read from remote", errors.New("exit status 128")
				}
				return "OK", nil
			},
		}

		err := FetchAndResetWithCmd(mock, tmpDir)
		if err == nil {
			t.Error("Expected error when fetch fails")
		}
	})

	t.Run("reset fails", func(t *testing.T) {
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				if args[0] == "reset" {
					return "fatal: could not reset", errors.New("exit status 1")
				}
				return "OK", nil
			},
		}

		err := FetchAndResetWithCmd(mock, tmpDir)
		if err == nil {
			t.Error("Expected error when reset fails")
		}
	})

	t.Run("clean fails", func(t *testing.T) {
		mock := &MockCommander{
			RunFunc: func(dir string, args ...string) (string, error) {
				if args[0] == "clean" {
					return "fatal: could not clean", errors.New("exit status 1")
				}
				return "OK", nil
			},
		}

		err := FetchAndResetWithCmd(mock, tmpDir)
		if err == nil {
			t.Error("Expected error when clean fails")
		}
	})
}
