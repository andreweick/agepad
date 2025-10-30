# Implementation Tasks

## Phase 1: Wizard Core (6 tasks)

- [ ] 1.1 Create `wizard/` package with `IsInteractive()` function using `golang.org/x/term.IsTerminal()` and CI env check
- [ ] 1.2 Implement `PromptForFile()` using Bubble Tea textinput component for file path prompting
- [ ] 1.3 Implement `PromptForIdentity()` using Bubble Tea list component with options: [Generate New, Specify Existing Path]
- [ ] 1.4 Implement `PromptForRecipients()` using Bubble Tea textinput component for recipients file path
- [ ] 1.5 Add `age.GenerateAndWriteIdentity(path)` function to generate identity using `age.GenerateX25519Identity()` and write with 0600 permissions
- [ ] 1.6 Add `age.AppendRecipient(recipientsPath, publicKey)` function to append public key to recipients file

## Phase 2: Wizard Integration (3 tasks)

- [ ] 2.1 Wire wizard into `cmd/agepad/main.go` in `runEditor()`:
  - Check `IsInteractive()` after flag parsing
  - Prompt for file path if `--file` not provided
  - Prompt for identity if `~/.config/age/key.txt` doesn't exist
  - Prompt for recipients if `.age-recipients` doesn't exist
- [ ] 2.2 Handle identity generation flow: display public key, offer to add to recipients file
- [ ] 2.3 Handle Ctrl+C gracefully in wizard (exit with status 130, no partial files)

## Phase 3: TUI Validation Toggle (3 tasks)

- [ ] 3.1 Add `validationEnabled bool` field to TUI model in `tui/tui.go`
- [ ] 3.2 Add Ctrl+V keybinding to toggle validation on/off
- [ ] 3.3 Display validation status in editor status bar:
  - When enabled: "✓ Valid JSON" or "✗ Invalid JSON: unexpected token at line 5"
  - When disabled: "Validation: OFF"
  - Run validator on content change when enabled

## Phase 4: Testing (3 tasks)

- [ ] 4.1 Test wizard flows:
  - Run `agepad` with no args and no defaults → wizard prompts appear
  - Run `agepad` with existing defaults → wizard skips
  - Test identity generation → creates file with correct permissions
  - Test non-interactive mode (piped input, CI=true) → no prompts, clear errors
- [ ] 4.2 Test TUI validation toggle:
  - Ctrl+V toggles validation on/off
  - Validation status updates in real-time
  - Saving works regardless of validation state
- [ ] 4.3 Manual testing in real terminal on macOS/Linux

## Phase 5: Documentation (2 tasks)

- [ ] 5.1 Update README.md:
  - Add section on interactive wizard
  - Document that wizard only appears when defaults missing
  - Show example of first-time user flow
  - Document Ctrl+V validation toggle
- [ ] 5.2 Update TUI help text to include Ctrl+V keybinding for validation toggle
