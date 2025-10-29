// agepad: Securely edit AGE-encrypted files entirely in memory (Bubble Tea TUI).
//
// Highlights:
// - Plaintext never touches disk; editing is in-process RAM via Bubble Tea textarea.
// - Default ASCII-armored output (disable with --armor=false).
// - Default identities: ~/.config/age/key.txt (friendly guidance if missing).
// - Diff-before-save (Ctrl+D to preview; double Ctrl+S to confirm write).
// - Syntax checks for .env, .json, .yaml/.yml, .toml before encrypting.
// - Read-only view mode (--view) for peek-only sessions.
// - Recipient "health" preflight: encrypt to memory and immediately decrypt with
//   your identities to catch lock-out risks before writing.
// - Batch rotate subcommand: re-encrypt *.age files under a tree to a new recipients set.
// - Crash guard: recover with a helpful message; buffer was only in RAM (never on disk).
// - Env-injection subcommand: `agepad run -- file.age -- cmd args...` exports KEY=VALs
//   from the decrypted file into the child process env without creating temp files.

package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	agepkg "github.com/andreweick/agepad/age"
	"github.com/andreweick/agepad/model"
	"github.com/andreweick/agepad/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/urfave/cli/v3"
)

const appName = "agepad"

const (
	defaultRecipientsFile = ".age-recipients"
)

func defaultIdentitiesPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "age", "key.txt")
}

func main() {
	cmd := &cli.Command{
		Name:  appName,
		Usage: "Securely edit AGE-encrypted files entirely in memory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "file",
				Usage:    "Path to the .age file to edit",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "recipients-file",
				Usage: "Path to recipients file",
				Value: defaultRecipientsFile,
			},
			&cli.StringFlag{
				Name:  "identities",
				Usage: "Path to AGE identities",
				Value: defaultIdentitiesPath(),
			},
			&cli.BoolFlag{
				Name:  "armor",
				Usage: "Write ASCII-armored .age output",
				Value: true,
			},
			&cli.BoolFlag{
				Name:  "view",
				Usage: "Open in read-only view mode (no edits)",
				Value: false,
			},
		},
		Action: runEditor,
		Commands: []*cli.Command{
			{
				Name:  "rotate",
				Usage: "Re-encrypt *.age files under a tree to a new recipients set",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "root",
						Usage: "Root directory to scan for *.age files",
						Value: ".",
					},
					&cli.StringFlag{
						Name:  "from",
						Usage: "Current recipients file (for logging/documentation)",
						Value: defaultRecipientsFile,
					},
					&cli.StringFlag{
						Name:     "to",
						Usage:    "NEW recipients file to use",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "identities",
						Usage: "AGE identities used to decrypt during rotation",
						Value: defaultIdentitiesPath(),
					},
				},
				Action: runRotate,
			},
			{
				Name:      "run",
				Usage:     "Export KEY=VALs from decrypted file into child process env",
				ArgsUsage: "-- <file.age> -- <command> [args...]",
				Action:    runEnvExec,
			},
		},
	}

	// Crash guard: keep messaging kind, remind that plaintext never hit disk.
	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintln(os.Stderr, "\n[CRASH-GUARD] The editor hit a fatal error.")
			fmt.Fprintln(os.Stderr, "Your edits were only in RAM; reopen the file and reapply recent changes.")
			os.Exit(3)
		}
	}()

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func runEditor(ctx context.Context, cmd *cli.Command) error {
	cfg := model.Config{
		FilePath:       cmd.String("file"),
		RecipientsFile: cmd.String("recipients-file"),
		IdentitiesPath: cmd.String("identities"),
		Armor:          cmd.Bool("armor"),
		ViewOnly:       cmd.Bool("view"),
	}

	// Friendly guidance if key missing
	if _, err := os.Stat(cfg.IdentitiesPath); err != nil {
		return fmt.Errorf("\nAGE private key not found at %s\n"+
			"- Generate one: age-keygen --output %s\n"+
			"- Or pass a different path: --identities /path/to/key.txt\n", cfg.IdentitiesPath, cfg.IdentitiesPath)
	}

	ids, err := agepkg.LoadIdentities(cfg.IdentitiesPath)
	if err != nil {
		return err
	}
	recips, err := agepkg.LoadRecipients(cfg.RecipientsFile)
	if err != nil {
		return err
	}
	plain, err := agepkg.DecryptToMemory(cfg.FilePath, ids)
	if err != nil {
		return err
	}

	m := tui.NewModel(cfg, plain, ids, recips)
	if err := tea.NewProgram(m, tea.WithAltScreen()).Start(); err != nil {
		return fmt.Errorf("tui error: %w", err)
	}
	return nil
}

func runRotate(ctx context.Context, cmd *cli.Command) error {
	cfg := model.RotateConfig{
		Root:               cmd.String("root"),
		FromRecipientsFile: cmd.String("from"),
		ToRecipientsFile:   cmd.String("to"),
		IdentitiesPath:     cmd.String("identities"),
	}

	ids, err := agepkg.LoadIdentities(cfg.IdentitiesPath)
	if err != nil {
		return err
	}
	newRecips, err := agepkg.LoadRecipients(cfg.ToRecipientsFile)
	if err != nil {
		return err
	}

	var files []string
	err = filepath.WalkDir(cfg.Root, func(path string, d fs.DirEntry, err error) error {
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
		return fmt.Errorf("rotate: no .age files found under %s", cfg.Root)
	}

	ok, fail := 0, 0
	for _, f := range files {
		plain, err := agepkg.DecryptToMemory(f, ids)
		if err != nil {
			fmt.Fprintf(os.Stderr, "rotate: decrypt failed for %s: %v\n", f, err)
			fail++
			continue
		}
		if err := agepkg.AtomicEncryptWrite(f, []byte(plain), newRecips, true /* keep armor on rotate */); err != nil {
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

func runEnvExec(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	// Syntax: agepad run -- <file.age> -- <command> [args...]
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
		return fmt.Errorf(`run usage: ` + appName + ` run -- <file.age> -- <command> [args...]`)
	}
	files := args[firstDash+1 : secondDash]
	if len(files) != 1 {
		return fmt.Errorf("run: expected exactly one AGE file after the first --")
	}
	runFile := files[0]
	runArgs := args[secondDash+1:]

	cfg := model.RunConfig{
		FilePath:       runFile,
		IdentitiesPath: defaultIdentitiesPath(),
		Command:        runArgs,
	}

	ids, err := agepkg.LoadIdentities(cfg.IdentitiesPath)
	if err != nil {
		return err
	}
	plain, err := agepkg.DecryptToMemory(cfg.FilePath, ids)
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
	cmdName := cfg.Command[0]
	path, err := exec.LookPath(cmdName)
	if err != nil {
		return fmt.Errorf("run: command not found: %s", cmdName)
	}
	return syscall.Exec(path, cfg.Command, newEnv)
}
