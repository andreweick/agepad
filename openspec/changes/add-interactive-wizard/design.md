# Interactive Wizard Design

## Context

agepad currently requires users to know all CLI flags and have default files in place. New users find this confusing when `~/.config/age/key.txt` or `.age-recipients` don't exist. Adding interactive Bubble Tea prompts when defaults are missing will lower the barrier to entry while maintaining full backward compatibility.

**Constraints:**
- Must not break existing flag-based usage
- Must work gracefully in non-interactive environments (CI/CD, pipes, scripts)
- Should use existing dependencies (Bubble Tea, age library)
- Keep it simple - only prompt for missing defaults

**Stakeholders:**
- New users who don't have identity/recipients files set up
- Existing users with scripts (must not break)
- CI/CD pipelines (must detect and skip prompts)

## Goals / Non-Goals

**Goals:**
- Make agepad usable for first-time users without consulting documentation
- Provide helpful descriptions explaining what each file is for
- Support identity generation without external tools
- Maintain 100% backward compatibility with existing flag usage
- Gracefully degrade in non-interactive environments

**Non-Goals:**
- Full-featured wizard with all options (only prompt for missing required config)
- Syntax highlighting or pretty-printing (skip for MVP)
- Armor mode prompting (always use armor)
- Saved preferences or configuration files (keep it stateless)
- Prompting for optional settings (validation is a TUI toggle, not a prompt)

## Decisions

### Decision 1: Use Bubble Tea for Wizard UI

**Choice**: Use `github.com/charmbracelet/bubbles` components (textinput, list)

**Rationale:**
- Already a dependency (used for main TUI editor)
- Consistent UX between wizard and editor
- Better user experience than standard library prompts
- Well-documented, actively maintained
- Users already familiar with Bubble Tea if they've used the editor

**Components to use:**
- `textinput` - for file path prompts
- `list` - for option selection (e.g., Generate vs Specify Existing)

### Decision 2: Use age Library Directly for Key Generation

**Choice**: Use `age.GenerateX25519Identity()` from `filippo.io/age` library

**Rationale:**
- Already a dependency
- No need for external `age-keygen` tool
- Aligns with single-binary compilation philosophy
- Better error handling (native Go errors vs parsing shell output)
- More reliable - guaranteed to work if agepad is installed
- Same code that `age-keygen` uses internally

**Implementation:**
```go
// Generate new identity
identity, err := age.GenerateX25519Identity()
if err != nil {
    return err
}

// Get private key (for identity file)
privateKey := identity.String() // "AGE-SECRET-KEY-1..."

// Get public key (for recipients file)
publicKey := identity.Recipient().String() // "age1..."

// Write to file with 0600 permissions
err = os.WriteFile(path, []byte(privateKey+"\n"), 0600)
```

### Decision 3: Validation Toggle in TUI, Not CLI

**Choice**: Add validation toggle in TUI editor (Ctrl+V), not as CLI flag or wizard prompt

**Rationale:**
- Users want to toggle validation during editing, not decide upfront
- Keeps CLI interface simple
- Natural workflow: start editing → see format → toggle validation if needed
- Doesn't interrupt wizard flow with unnecessary questions

**Implementation:**
- Add `validationEnabled bool` to TUI model
- Bind Ctrl+V to toggle
- Show validation status in status bar
- Run validator on content change when enabled

### Decision 4: TTY Detection Strategy

**Choice**: Use `golang.org/x/term.IsTerminal()` + check `CI` environment variable

**Rationale:**
- `term.IsTerminal()` is the standard Go approach for TTY detection
- Checking `CI=true` is a common convention (GitHub Actions, GitLab CI, etc.)
- Simple and reliable

**Implementation:**
```go
func IsInteractive() bool {
    if os.Getenv("CI") == "true" {
        return false
    }
    return term.IsTerminal(int(os.Stdin.Fd()))
}
```

### Decision 5: Prompt Placement

**Choice**: Prompt **after** flag parsing, **before** loading files

**Rationale:**
- Flags take precedence (explicit beats implicit)
- Prompts only fill in missing values
- Clear separation: parse flags → prompt for missing → load/validate → run

**Flow:**
```
1. Parse flags with urfave/cli
2. Check IsInteractive()
3. If interactive and file missing: prompt for file path
4. If interactive and identity missing: prompt (generate or specify)
5. If interactive and recipients missing: prompt for path
6. Load and validate all configuration
7. Run editor/command
```

### Decision 6: Always Use Armor

**Choice**: Remove armor flag prompting - always use ASCII armor output

**Rationale:**
- Armor is the better default (readable, git-friendly)
- Removes complexity from wizard
- Users who want binary can use `--armor=false` flag
- No prompt needed

## Risks / Trade-offs

### Risk: Users confused by prompts in scripts

**Mitigation:**
- Robust TTY detection ensures prompts don't appear in non-interactive contexts
- Test thoroughly with pipes, redirects, and CI environments
- Document non-interactive behavior clearly in README

### Risk: Bubble Tea dependency size

**Impact**: Minimal - already a dependency for the editor
**Acceptance**: No additional cost

### Risk: Identity generation without review

**Mitigation:**
- Display generated public key to user
- Offer to add to recipients file
- User must confirm before proceeding

### Trade-off: Slightly longer startup time for interactive mode

**Impact**: Minimal - prompts only shown when config is missing
**Acceptance**: Worth it for improved first-time user experience

## Migration Plan

**Phase 1: Wizard Implementation**
1. Create `wizard/` package with Bubble Tea prompts
2. Implement file path, identity, and recipients prompting
3. Wire into main.go before editor launch
4. Test in interactive and non-interactive modes

**Phase 2: TUI Validation Toggle**
5. Add validation toggle state to TUI model
6. Bind Ctrl+V keybinding
7. Show validation status in editor
8. Update help text

**Phase 3: Polish**
9. Improve error messages and validation feedback
10. Add comprehensive tests
11. Update README with examples

**Rollback:**
- If critical bugs emerge, add `--no-wizard` flag to disable all prompts
- Doesn't break existing users since wizard is additive (only appears when defaults missing)

## Implementation Notes

### Package Structure

```
wizard/
├── wizard.go          # Main wizard orchestration
├── prompts.go         # Individual Bubble Tea prompt models
└── interactive.go     # TTY detection logic

age/
├── age.go            # Existing encryption functions
└── keygen.go         # New: identity generation and file writing
```

### Key Functions

```go
// wizard package
func IsInteractive() bool
func PromptForFile() (string, error)
func PromptForIdentity() (string, error) // returns path to identity file
func PromptForRecipients() (string, error)

// age package additions
func GenerateAndWriteIdentity(path string) (*age.X25519Identity, error)
func AppendRecipient(recipientsPath, publicKey string) error
```

### Testing Strategy

- **Unit tests**: Test wizard functions with mocked TTY
- **Integration tests**: Test full CLI flows with simulated input
- **Manual testing**: Real terminal testing on macOS, Linux
- **CI testing**: Ensure prompts don't break CI pipelines (CI=true test)

## Open Questions

**Q: Should validation be on or off by default in the editor?**
- **Answer**: OFF by default - less intrusive, user can enable if needed

**Q: Should we support other identity types (SSH keys, scrypt)?**
- **Answer**: No - X25519 only for MVP, keeps it simple

**Q: Should generated identity path be configurable?**
- **Answer**: No - always use ~/.config/age/key.txt for generated keys, users can specify custom path via "Specify Existing" option
