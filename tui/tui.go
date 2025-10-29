package tui

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/andreweick/agepad/model"
	agepkg "github.com/andreweick/agepad/age"
	"github.com/andreweick/agepad/validator"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pmezard/go-difflib/difflib"
)

// Model represents the TUI editor state.
type Model struct {
	cfg        model.Config
	ta         textarea.Model
	orig       string // original plaintext (for diff)
	status     string
	err        error
	identities []age.Identity
	recips     []age.Recipient
	changed    bool
	savedAt    time.Time

	// Crash guard (RAM only)
	lastSnapshot string

	// Save confirmation
	pendingConfirm bool
}

type snapshotTick struct{}

// NewModel creates a new TUI model.
func NewModel(cfg model.Config, plaintext string, ids []age.Identity, recips []age.Recipient) Model {
	ta := textarea.New()
	ta.SetValue(plaintext)
	ta.Focus()
	ta.Placeholder = "Edit secrets…"
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.SetWidth(100)
	ta.SetHeight(30)
	if cfg.ViewOnly {
		ta.Blur()
	}

	m := Model{
		cfg:          cfg,
		ta:           ta,
		orig:         plaintext,
		status:       fmt.Sprintf("Opened %s (RAM). Ctrl+D: diff  Ctrl+S: save  Ctrl+Q: quit", cfg.FilePath),
		identities:   ids,
		recips:       recips,
		lastSnapshot: plaintext,
	}
	return m
}

// Init initializes the TUI model.
func (m Model) Init() tea.Cmd {
	// Periodic in-memory snapshot (no disk) for crash guard messaging.
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return snapshotTick{} })
}

// Update handles TUI events.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case snapshotTick:
		m.lastSnapshot = m.ta.Value()
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return snapshotTick{} })

	case tea.KeyMsg:
		switch t.String() {
		case "ctrl+q", "esc":
			// Double press protection if there are unsaved changes and not view-only
			if m.changed && !m.cfg.ViewOnly && !m.pendingConfirm {
				m.status = "Unsaved changes; press Ctrl+Q again to quit without saving"
				m.pendingConfirm = true
				return m, nil
			}
			return m, tea.Quit

		case "ctrl+d":
			diff := unifiedDiff(m.orig, m.ta.Value(), filepath.Base(m.cfg.FilePath))
			if strings.TrimSpace(diff) == "" {
				m.status = "No changes to show (buffers identical)."
			} else {
				m.status = "Diff preview (first 2000 chars):\n" + truncate(diff, 2000)
			}
			m.pendingConfirm = false
			return m, nil

		case "ctrl+s":
			if m.cfg.ViewOnly {
				m.status = "View-only mode: saving disabled."
				return m, nil
			}
			buf := m.ta.Value()

			// 1) Validate format (fail early before encryption)
			if err := validator.ValidateByExt(m.cfg.FilePath, buf); err != nil {
				m.err = err
				m.status = "Validation failed; not saved."
				m.pendingConfirm = false
				return m, nil
			}

			// 2) Recipient health preflight: encrypt to memory, then decrypt with identities.
			cipher, err := agepkg.EncryptToMemory([]byte(buf), m.recips, m.cfg.Armor)
			if err != nil {
				m.err = fmt.Errorf("preflight encrypt: %w", err)
				m.status = "Save aborted."
				m.pendingConfirm = false
				return m, nil
			}
			r, err := age.Decrypt(bytes.NewReader(cipher), m.identities...)
			if err != nil {
				m.err = fmt.Errorf("preflight decrypt failed with current identities; "+
					"you may lock yourself out: %w", err)
				m.status = "Save aborted. Update recipients or identities."
				m.pendingConfirm = false
				return m, nil
			}
			_, _ = io.ReadAll(r) // Drain; we only care that decryption is possible.

			// 3) Require explicit confirmation if content changed (double Ctrl+S).
			if m.ta.Value() != m.orig && !m.pendingConfirm {
				diff := unifiedDiff(m.orig, m.ta.Value(), filepath.Base(m.cfg.FilePath))
				m.status = "About to save. Diff (first 2000 chars):\n" +
					truncate(diff, 2000) + "\nPress Ctrl+S again to confirm."
				m.pendingConfirm = true
				return m, nil
			}

			// 4) Write atomically.
			if err := agepkg.AtomicEncryptWrite(m.cfg.FilePath, []byte(buf), m.recips, m.cfg.Armor); err != nil {
				m.err = err
				m.status = "Save failed"
			} else {
				m.err = nil
				m.savedAt = time.Now()
				m.status = fmt.Sprintf("Saved %s (armor=%v) at %s",
					m.cfg.FilePath, m.cfg.Armor, m.savedAt.Format(time.RFC3339))
				m.orig = buf
				m.changed = false
			}
			m.pendingConfirm = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.ta.Value()
	m.ta, cmd = m.ta.Update(msg)
	if prev != m.ta.Value() {
		m.changed = true
		m.pendingConfirm = false
	}
	return m, cmd
}

// View renders the TUI.
func (m Model) View() string {
	errLine := ""
	if m.err != nil {
		errLine = "\n[ERROR] " + m.err.Error()
	}
	return fmt.Sprintf("%s\n\n%s\n%s\n", m.status, m.ta.View(), errLine)
}

func unifiedDiff(a, b, filename string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(a),
		B:        difflib.SplitLines(b),
		FromFile: filename + " (original)",
		ToFile:   filename + " (edited)",
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	return text
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n…(truncated)…"
}
