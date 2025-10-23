// agepad: Securely edit AGE-encrypted files entirely in memory (Bubble Tea TUI).
//
// Highlights:
// - Plaintext never touches disk; editing is in-process RAM via Bubble Tea textarea.
// - Default ASCII-armored output (disable with --armor=false).
// - Default identities: ~/.config/age/key.txt (friendly guidance if missing).
// - Diff-before-save (Ctrl+D to preview; double Ctrl+S to confirm write).
// - Syntax checks for .env, .json, .yaml/.yml, .toml before encrypting.
// - Read-only view mode (--view) for peek-only sessions.
// - Recipient “health” preflight: encrypt to memory and immediately decrypt with
//   your identities to catch lock-out risks before writing.
// - Batch rotate subcommand: re-encrypt *.age files under a tree to a new recipients set.
// - Crash guard: recover with a helpful message; buffer was only in RAM (never on disk).
// - Env-injection subcommand: `agepad run -- file.age -- cmd args...` exports KEY=VALs
//   from the decrypted file into the child process env without creating temp files.
//
// Build:
//   go mod init github.com/andreweick/agepad
//   go get filippo.io/age github.com/charmbracelet/bubbletea github.com/charmbracelet/bubbles \
//          github.com/spf13/pflag github.com/pmezard/go-difflib/difflib \
//          gopkg.in/yaml.v3 github.com/pelletier/go-toml/v2
//   go build -o agepad
//
// TUI usage:
//   ./agepad --file secrets/app.env.age --recipients-file .age-recipients
//   ./agepad --file secrets/app.env.age --recipients-file .age-recipients --view
//
// Rotate recipients:
//   ./agepad rotate --root secrets --from .age-recipients --to .age-recipients.new --identities ~/.config/age/key.txt
//
// Env injection:
//   ./agepad run -- secrets/app.env.age -- myserver --port 8080
//
// Notes:
// - Long options only, by design.
// - Keep your repo-pinned .age-recipients for reproducibility and CI friendliness.

package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"filippo.io/age"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pelletier/go-toml/v2"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

const appName = "agepad"

const (
	defaultRecipientsFile = ".age-recipients"
)

// ----- Config & Flag Parsing -----

type config struct {
	// TUI / normal mode
	filePath       string
	recipientsFile string
	identitiesPath string
	armor          bool
	viewOnly       bool

	// Subcommand dispatch
	subcommand string

	// rotate subcommand
	rotateRoot       string
	rotateFromRecips string
	rotateToRecips   string
	identitiesRotate string

	// run subcommand
	runFile string
	runArgs []string
}

func defaultIdentitiesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "age", "key.txt")
}

func parseTopLevelFlags() (config, error) {
	var c config
	// Subcommand detection (we support: rotate, run)
	if len(os.Args) > 1 && (os.Args[1] == "rotate" || os.Args[1] == "run") {
		c.subcommand = os.Args[1]
		return c, nil
	}

	pflag.StringVar(&c.filePath, "file", "", "Path to the .age file to edit")
	pflag.StringVar(&c.recipientsFile, "recipients-file", defaultRecipientsFile, "Path to recipients file")
	pflag.StringVar(&c.identitiesPath, "identities", defaultIdentitiesPath(), "Path to AGE identities (default: ~/.config/age/key.txt)")
	pflag.BoolVar(&c.armor, "armor", true, "Write ASCII-armored .age output")
	pflag.BoolVar(&c.viewOnly, "view", false, "Open in read-only view mode (no edits)")
	pflag.Parse()

	if c.filePath == "" {
		return c, errors.New("--file is required (or use subcommands: rotate | run)")
	}
	return c, nil
}

func parseRotateFlags() (config, error) {
	var c config
	c.subcommand = "rotate"
	fs := pflag.NewFlagSet("rotate", pflag.ContinueOnError)
	fs.StringVar(&c.rotateRoot, "root", ".", "Root directory to scan for *.age files")
	fs.StringVar(&c.rotateFromRecips, "from", defaultRecipientsFile, "Current recipients file (for logging/documentation)")
	fs.StringVar(&c.rotateToRecips, "to", "", "NEW recipients file to use (required)")
	fs.StringVar(&c.identitiesRotate, "identities", defaultIdentitiesPath(), "AGE identities used to decrypt during rotation")
	if err := fs.Parse(os.Args[2:]); err != nil {
		return c, err
	}
	if c.rotateToRecips == "" {
		return c, errors.New("rotate: --to is required (path to new recipients file)")
	}
	return c, nil
}

func parseRunFlags() (config, error) {
	var c config
	c.subcommand = "run"
	// Syntax: agepad run -- <file.age> -- <command> [args...]
	args := os.Args[2:]
	firstDash, secondDash := -1, -1
	for i, a := range args {
		if a == "--" {
			if firstDash == -1 {
				firstDash = i
			} else {
				secondDash = i
				break
			}
		}
	}
	if firstDash == -1 || secondDash == -1 || secondDash == len(args)-1 {
		return c, errors.New(`run usage: ` + appName + ` run -- <file.age> -- <command> [args...]`)
	}
	files := args[firstDash+1 : secondDash]
	if len(files) != 1 {
		return c, errors.New("run: expected exactly one AGE file after the first --")
	}
	c.runFile = files[0]
	c.runArgs = args[secondDash+1:]
	return c, nil
}

// ----- AGE Helpers -----

func loadIdentities(path string) ([]age.Identity, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("\nCould not read AGE key at %s\n"+
			"- If you don’t have one:   age-keygen --output %s\n"+
			"- Or point to another key: --identities /path/to/key.txt\nOriginal error: %w",
			path, path, err)
	}
	ids, err := age.ParseIdentities(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse identities in %s: %w", path, err)
	}
	return ids, nil
}

func loadRecipients(path string) ([]age.Recipient, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("\nRecipients file not found: %s\n"+
			"- Create one and commit it to your repo (recommended).\n"+
			"- Example (one public key per line): age1xxxx...\nOriginal error: %w", path, err)
	}
	rs, err := age.ParseRecipients(bytes.NewReader(b))
	if err != nil {
		return nil, fmt.Errorf("failed to parse recipients in %s: %w", path, err)
	}
	if len(rs) == 0 {
		return nil, fmt.Errorf("no recipients in %s; add at least one age public key", path)
	}
	return rs, nil
}

func decryptToMemory(cipherPath string, ids []age.Identity) (string, error) {
	f, err := os.Open(cipherPath)
	if err != nil {
		return "", fmt.Errorf("open ciphertext: %w", err)
	}
	defer f.Close()
	r, err := age.Decrypt(f, ids...)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("read plaintext: %w", err)
	}
	return string(plain), nil
}

func encryptToMemory(plaintext []byte, recips []age.Recipient, armor bool) ([]byte, error) {
	var buf bytes.Buffer
	if armor {
		aw, err := age.Armor(&buf)
		if err != nil {
			return nil, err
		}
		w, err := age.Encrypt(aw, recips...)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(plaintext); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil { // closes the encryption stream
			return nil, err
		}
		if err := aw.Close(); err != nil { // closes the armor wrapper
			return nil, err
		}
		return buf.Bytes(), nil
	}
	w, err := age.Encrypt(&buf, recips...)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func atomicEncryptWrite(dstPath string, b []byte, recips []age.Recipient, armor bool) error {
	dir := filepath.Dir(dstPath)
	tmp, err := os.CreateTemp(dir, ".agepad-tmp-*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if armor {
		aw, err := age.Armor(tmp)
		if err != nil {
			return fmt.Errorf("armor: %w", err)
		}
		w, err := age.Encrypt(aw, recips...)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		if err := aw.Close(); err != nil {
			return err
		}
	} else {
		w, err := age.Encrypt(tmp, recips...)
		if err != nil {
			return fmt.Errorf("encrypt: %w", err)
		}
		if _, err := w.Write(b); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	return os.Rename(tmpPath, dstPath) // atomic replace on same filesystem
}

// ----- Format Validation -----

func validateByExt(filename string, content string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".json":
		var v any
		dec := json.NewDecoder(strings.NewReader(content))
		dec.UseNumber()
		if err := dec.Decode(&v); err != nil {
			return fmt.Errorf("JSON parse error: %w", err)
		}
		return nil
	case ".yaml", ".yml":
		var v any
		if err := yaml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("YAML parse error: %w", err)
		}
		return nil
	case ".toml":
		var v any
		if err := toml.Unmarshal([]byte(content), &v); err != nil {
			return fmt.Errorf("TOML parse error: %w", err)
		}
		return nil
	default:
		// If it looks like .env, validate basic KEY=VAL lines; otherwise accept.
		if looksLikeDotEnv(content) {
			if err := validateDotEnv(content); err != nil {
				return err
			}
		}
		return nil
	}
}

func looksLikeDotEnv(s string) bool {
	sc := bufio.NewScanner(strings.NewReader(s))
	lines, matches := 0, 0
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines++
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "=") {
			matches++
		}
	}
	return lines > 0 && matches > 0
}

func validateDotEnv(s string) error {
	sc := bufio.NewScanner(strings.NewReader(s))
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := sc.Text()
		t := strings.TrimSpace(line)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		if !strings.Contains(t, "=") || strings.HasPrefix(t, "=") {
			return fmt.Errorf(".env parse error on line %d: expected KEY=VALUE", lineNo)
		}
		kv := strings.SplitN(t, "=", 2)
		key := strings.TrimSpace(kv[0])
		if key == "" || strings.ContainsAny(key, " \t\"'") {
			return fmt.Errorf(".env invalid key on line %d", lineNo)
		}
	}
	return nil
}

// ----- Diff Helper -----

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

// ----- TUI Model -----

type model struct {
	cfg        config
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

func initialModel(cfg config) (model, error) {
	m := model{cfg: cfg}

	// Friendly guidance if key missing
	if _, err := os.Stat(cfg.identitiesPath); err != nil {
		return m, fmt.Errorf("\nAGE private key not found at %s\n"+
			"- Generate one: age-keygen --output %s\n"+
			"- Or pass a different path: --identities /path/to/key.txt\n", cfg.identitiesPath, cfg.identitiesPath)
	}

	ids, err := loadIdentities(cfg.identitiesPath)
	if err != nil {
		return m, err
	}
	recips, err := loadRecipients(cfg.recipientsFile)
	if err != nil {
		return m, err
	}
	plain, err := decryptToMemory(cfg.filePath, ids)
	if err != nil {
		return m, err
	}

	ta := textarea.New()
	ta.SetValue(plain)
	ta.Focus()
	ta.Placeholder = "Edit secrets…"
	ta.ShowLineNumbers = true
	ta.CharLimit = 0
	ta.SetWidth(100)
	ta.SetHeight(30)
	if cfg.viewOnly {
		ta.Blur()
	}

	m = model{
		cfg:          cfg,
		ta:           ta,
		orig:         plain,
		status:       fmt.Sprintf("Opened %s (RAM). Ctrl+D: diff  Ctrl+S: save  Ctrl+Q: quit", cfg.filePath),
		identities:   ids,
		recips:       recips,
		lastSnapshot: plain,
	}
	return m, nil
}

func (m model) Init() tea.Cmd {
	// Periodic in-memory snapshot (no disk) for crash guard messaging.
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg { return snapshotTick{} })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch t := msg.(type) {
	case snapshotTick:
		m.lastSnapshot = m.ta.Value()
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return snapshotTick{} })

	case tea.KeyMsg:
		switch t.String() {
		case "ctrl+q", "esc":
			// Double press protection if there are unsaved changes and not view-only
			if m.changed && !m.cfg.viewOnly && !m.pendingConfirm {
				m.status = "Unsaved changes; press Ctrl+Q again to quit without saving"
				m.pendingConfirm = true
				return m, nil
			}
			return m, tea.Quit

		case "ctrl+d":
			diff := unifiedDiff(m.orig, m.ta.Value(), filepath.Base(m.cfg.filePath))
			if strings.TrimSpace(diff) == "" {
				m.status = "No changes to show (buffers identical)."
			} else {
				m.status = "Diff preview (first 2000 chars):\n" + truncate(diff, 2000)
			}
			m.pendingConfirm = false
			return m, nil

		case "ctrl+s":
			if m.cfg.viewOnly {
				m.status = "View-only mode: saving disabled."
				return m, nil
			}
			buf := m.ta.Value()

			// 1) Validate format (fail early before encryption)
			if err := validateByExt(m.cfg.filePath, buf); err != nil {
				m.err = err
				m.status = "Validation failed; not saved."
				m.pendingConfirm = false
				return m, nil
			}

			// 2) Recipient health preflight: encrypt to memory, then decrypt with identities.
			cipher, err := encryptToMemory([]byte(buf), m.recips, m.cfg.armor)
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
				diff := unifiedDiff(m.orig, m.ta.Value(), filepath.Base(m.cfg.filePath))
				m.status = "About to save. Diff (first 2000 chars):\n" +
					truncate(diff, 2000) + "\nPress Ctrl+S again to confirm."
				m.pendingConfirm = true
				return m, nil
			}

			// 4) Write atomically.
			if err := atomicEncryptWrite(m.cfg.filePath, []byte(buf), m.recips, m.cfg.armor); err != nil {
				m.err = err
				m.status = "Save failed"
			} else {
				m.err = nil
				m.savedAt = time.Now()
				m.status = fmt.Sprintf("Saved %s (armor=%v) at %s",
					m.cfg.filePath, m.cfg.armor, m.savedAt.Format(time.RFC3339))
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

func (m model) View() string {
	errLine := ""
	if m.err != nil {
		errLine = "\n[ERROR] " + m.err.Error()
	}
	return fmt.Sprintf("%s\n\n%s\n%s\n", m.status, m.ta.View(), errLine)
}

// ----- Subcommand: rotate recipients -----

func runRotate(cfg config) error {
	ids, err := loadIdentities(cfg.identitiesRotate)
	if err != nil {
		return err
	}
	newRecips, err := loadRecipients(cfg.rotateToRecips)
	if err != nil {
		return err
	}

	var files []string
	err = filepath.WalkDir(cfg.rotateRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(d.Name()), ".age") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("rotate: no .age files found under %s", cfg.rotateRoot)
	}

	ok, fail := 0, 0
	for _, f := range files {
		plain, err := decryptToMemory(f, ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "rotate: decrypt failed for %s: %v\n", f, err)
			fail++
			continue
		}
		if err := atomicEncryptWrite(f, []byte(plain), newRecips, true /* keep armor on rotate */); err != nil {
			fmt.Fprintf(os.Stderr, "rotate: re-encrypt failed for %s: %v\n", f, err)
			fail++
			continue
		}
		ok++
	}
	fmt.Printf("rotate complete: %d success, %d failed\n", ok, fail)
	if fail > 0 {
		return fmt.Errorf("rotate: some files failed (see stderr)")
	}
	return nil
}

// ----- Subcommand: run (env inject + exec) -----

func runEnvExec(cfg config) error {
	ids, err := loadIdentities(defaultIdentitiesPath())
	if err != nil {
		return err
	}
	plain, err := decryptToMemory(cfg.runFile, ids)
	if err != nil {
		return err
	}

	// Merge decrypted KEY=VAL lines into environment (simple .env semantics).
	envMap := map[string]string{}
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	sc := bufio.NewScanner(strings.NewReader(plain))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		key := strings.TrimSpace(parts[0])
		val := parts[1] // keep raw; allow spaces
		if key != "" {
			envMap[key] = val
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Convert to []string form for Exec
	var newEnv []string
	for k, v := range envMap {
		newEnv = append(newEnv, k+"="+v)
	}

	// Replace current process with target command (posix-style)
	cmd := cfg.runArgs[0]
	path, err := exec.LookPath(cmd)
	if err != nil {
		return fmt.Errorf("run: command not found: %s", cmd)
	}
	return syscall.Exec(path, cfg.runArgs, newEnv)
}

// ----- Main -----

func main() {
	// Subcommands first
	if len(os.Args) > 1 && os.Args[1] == "rotate" {
		cfg, err := parseRotateFlags()
		if err != nil {
			fmt.Fprintln(os.Stderr, "rotate:", err)
			os.Exit(2)
		}
		if err := runRotate(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "rotate:", err)
			os.Exit(1)
		}
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "run" {
		cfg, err := parseRunFlags()
		if err != nil {
			fmt.Fprintln(os.Stderr, "run:", err)
			os.Exit(2)
		}
		if err := runEnvExec(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "run:", err)
			os.Exit(1)
		}
		return
	}

	// Normal TUI mode
	cfg, err := parseTopLevelFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}

	// Crash guard: keep messaging kind, remind that plaintext never hit disk.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "\n[CRASH-GUARD] The editor hit a fatal error.")
			fmt.Fprintln(os.Stderr, "Your edits were only in RAM; reopen the file and reapply recent changes.")
			os.Exit(3)
		}
	}()

	m, err := initialModel(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, "init error:", err)
		os.Exit(1)
	}
	if err := tea.NewProgram(m, tea.WithAltScreen()).Start(); err != nil {
		fmt.Fprintln(os.Stderr, "tui error:", err)
		os.Exit(1)
	}
}