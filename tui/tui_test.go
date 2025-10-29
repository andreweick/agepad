package tui

import (
	"fmt"
	"testing"

	"filippo.io/age"
	"github.com/andreweick/agepad/model"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	t.Run("creates model with provided configuration", func(t *testing.T) {
		cfg := model.Config{
			FilePath:       "/path/to/test.age",
			RecipientsFile: ".age-recipients",
			IdentitiesPath: "~/.config/age/key.txt",
			Armor:          true,
			ViewOnly:       false,
		}

		identity, err := age.GenerateX25519Identity()
		if err != nil {
			t.Fatalf("failed to generate identity: %v", err)
		}
		recipient := identity.Recipient()

		plaintext := "test content"
		m := NewModel(cfg, plaintext, []age.Identity{identity}, []age.Recipient{recipient})

		if m.cfg.FilePath != cfg.FilePath {
			t.Errorf("expected FilePath %s, got %s", cfg.FilePath, m.cfg.FilePath)
		}
		if m.orig != plaintext {
			t.Errorf("expected orig to be %q, got %q", plaintext, m.orig)
		}
		if m.ta.Value() != plaintext {
			t.Errorf("expected textarea value to be %q, got %q", plaintext, m.ta.Value())
		}
		if len(m.identities) != 1 {
			t.Errorf("expected 1 identity, got %d", len(m.identities))
		}
		if len(m.recips) != 1 {
			t.Errorf("expected 1 recipient, got %d", len(m.recips))
		}
	})

	t.Run("creates view-only model with blurred textarea", func(t *testing.T) {
		cfg := model.Config{
			FilePath: "/path/to/test.age",
			ViewOnly: true,
		}

		plaintext := "view only content"
		m := NewModel(cfg, plaintext, nil, nil)

		if !m.cfg.ViewOnly {
			t.Error("expected ViewOnly to be true")
		}
		if m.ta.Focused() {
			t.Error("expected textarea to be blurred in view-only mode")
		}
	})
}

func TestModelUpdate(t *testing.T) {
	t.Run("marks content as changed when textarea is edited", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)

		if m.changed {
			t.Error("expected changed to be false initially")
		}

		// Simulate typing
		m.ta.SetValue("modified")
		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		m = result.(Model)

		if !m.changed {
			t.Error("expected changed to be true after edit")
		}
	})

	t.Run("shows quit confirmation on ctrl+q with unsaved changes", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)
		m.changed = true

		result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})
		m = result.(Model)

		if cmd != nil {
			t.Error("expected no quit command on first ctrl+q with unsaved changes")
		}
		if !m.pendingConfirm {
			t.Error("expected pendingConfirm to be true after first ctrl+q")
		}
	})

	t.Run("quits immediately on ctrl+q with no changes", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)

		_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlQ})

		if cmd == nil {
			t.Error("expected quit command on ctrl+q with no changes")
		}
	})

	t.Run("shows diff on ctrl+d", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)
		m.ta.SetValue("modified")

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		m = result.(Model)

		if m.status == "" {
			t.Error("expected status to be updated with diff")
		}
	})

	t.Run("shows no changes message when diff is empty", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
		m = result.(Model)

		if m.status != "No changes to show (buffers identical)." {
			t.Errorf("expected no changes message, got: %s", m.status)
		}
	})

	t.Run("prevents saving in view-only mode", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age", ViewOnly: true}
		m := NewModel(cfg, "original", nil, nil)

		result, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
		m = result.(Model)

		if m.status != "View-only mode: saving disabled." {
			t.Errorf("expected view-only message, got: %s", m.status)
		}
	})

	t.Run("updates snapshot on snapshot tick", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "original", nil, nil)
		m.ta.SetValue("new content")

		result, _ := m.Update(snapshotTick{})
		m = result.(Model)

		if m.lastSnapshot != "new content" {
			t.Errorf("expected lastSnapshot to be updated to 'new content', got %q", m.lastSnapshot)
		}
	})
}

func TestView(t *testing.T) {
	t.Run("renders view with status and textarea", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "content", nil, nil)

		view := m.View()

		if view == "" {
			t.Error("expected non-empty view")
		}
		if len(view) < 10 {
			t.Error("expected view to contain substantial content")
		}
	})

	t.Run("includes error in view when present", func(t *testing.T) {
		cfg := model.Config{FilePath: "test.age"}
		m := NewModel(cfg, "content", nil, nil)
		m.err = fmt.Errorf("test error")

		view := m.View()

		if !contains(view, "[ERROR]") {
			t.Error("expected view to contain [ERROR] when error is present")
		}
		if !contains(view, "test error") {
			t.Error("expected view to contain error message")
		}
	})
}

func TestUnifiedDiff(t *testing.T) {
	t.Run("generates diff for different strings", func(t *testing.T) {
		a := "line1\nline2\nline3"
		b := "line1\nmodified\nline3"

		diff := unifiedDiff(a, b, "test.txt")

		if diff == "" {
			t.Error("expected non-empty diff")
		}
		if !contains(diff, "test.txt") {
			t.Error("expected diff to contain filename")
		}
	})

	t.Run("generates empty diff for identical strings", func(t *testing.T) {
		a := "same content"
		b := "same content"

		diff := unifiedDiff(a, b, "test.txt")

		if diff != "" {
			t.Errorf("expected empty diff for identical strings, got: %s", diff)
		}
	})
}

func TestTruncate(t *testing.T) {
	t.Run("truncates string longer than limit", func(t *testing.T) {
		s := "this is a very long string that should be truncated"
		truncated := truncate(s, 10)

		if len(truncated) > 30 {
			t.Errorf("expected truncated string to be around 10 chars + marker, got %d", len(truncated))
		}
		if !contains(truncated, "truncated") {
			t.Error("expected truncation marker in output")
		}
	})

	t.Run("does not truncate string shorter than limit", func(t *testing.T) {
		s := "short"
		truncated := truncate(s, 100)

		if truncated != s {
			t.Errorf("expected string to remain unchanged, got %q", truncated)
		}
	})

	t.Run("does not truncate string equal to limit", func(t *testing.T) {
		s := "exactly10c"
		truncated := truncate(s, 10)

		if truncated != s {
			t.Errorf("expected string to remain unchanged, got %q", truncated)
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
