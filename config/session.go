// Package config provides session metadata for worktrees.
package config

import (
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// SessionMetadata stores information about how a worktree was created.
type SessionMetadata struct {
	// Creation info
	CreatedAt time.Time `yaml:"created_at"`
	Repo      string    `yaml:"repo"`
	Branch    string    `yaml:"branch"`
	Prompt    string    `yaml:"prompt"`

	// Preset configuration
	PresetName string   `yaml:"preset_name"`
	Agents     []string `yaml:"agents"`

	// Runtime info
	LastOpened time.Time `yaml:"last_opened,omitempty"`
}

// SessionMetadataFileName is the name of the metadata file in each worktree.
const SessionMetadataFileName = ".orchestrate-session.yaml"

// LoadSessionMetadata loads session metadata from a worktree directory.
func LoadSessionMetadata(worktreePath string) (*SessionMetadata, error) {
	metaPath := filepath.Join(worktreePath, SessionMetadataFileName)

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, err
	}

	var meta SessionMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// SaveSessionMetadata saves session metadata to a worktree directory.
func SaveSessionMetadata(worktreePath string, meta *SessionMetadata) error {
	metaPath := filepath.Join(worktreePath, SessionMetadataFileName)

	data, err := yaml.Marshal(meta)
	if err != nil {
		return err
	}

	return os.WriteFile(metaPath, data, 0644)
}

// CreateSessionMetadata creates a new session metadata with the given info.
func CreateSessionMetadata(repo, branch, prompt, presetName string, agents []string) *SessionMetadata {
	return &SessionMetadata{
		CreatedAt:  time.Now(),
		Repo:       repo,
		Branch:     branch,
		Prompt:     prompt,
		PresetName: presetName,
		Agents:     agents,
	}
}

