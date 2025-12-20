package keys

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestKeysShortHelp(t *testing.T) {
	help := Keys.ShortHelp()
	if len(help) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestKeysFullHelp(t *testing.T) {
	help := Keys.FullHelp()
	if len(help) == 0 {
		t.Error("FullHelp should return binding groups")
	}
	for i, group := range help {
		if len(group) == 0 {
			t.Errorf("FullHelp group %d should not be empty", i)
		}
	}
}

func TestNavigationKeys(t *testing.T) {
	navKeys := NavigationKeys()
	if len(navKeys) != 4 {
		t.Errorf("NavigationKeys should return 4 keys, got %d", len(navKeys))
	}
}

func TestActionKeys(t *testing.T) {
	actionKeys := ActionKeys()
	if len(actionKeys) != 4 {
		t.Errorf("ActionKeys should return 4 keys, got %d", len(actionKeys))
	}
}

func TestViewKeys(t *testing.T) {
	// ViewKeys removed as they are now global Tab navigation
}

func TestKeyBindingsHaveHelp(t *testing.T) {
	bindings := []key.Binding{
		Keys.Up, Keys.Down, Keys.Left, Keys.Right,
		Keys.Enter, Keys.Back, Keys.Tab, Keys.Save,
		Keys.Help, Keys.Quit, Keys.Refresh, Keys.Delete,
	}

	for _, b := range bindings {
		h := b.Help()
		if h.Key == "" {
			t.Error("Key binding should have help key")
		}
		if h.Desc == "" {
			t.Error("Key binding should have help description")
		}
	}
}
