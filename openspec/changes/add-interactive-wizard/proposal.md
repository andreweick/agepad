# Add Interactive CLI Wizard

## Why
The current CLI requires users to know all the correct flags and file paths upfront, creating a steep learning curve for new users. When default files don't exist (identity, recipients), users must consult documentation or guess. An interactive wizard that prompts for missing defaults would make agepad accessible to first-time users while maintaining full backward compatibility.

## What Changes
- Add interactive Bubble Tea prompts when default files are missing
- Prompt for file path if `--file` flag not provided
- Prompt for identity if `~/.config/age/key.txt` doesn't exist (with option to generate new key)
- Prompt for recipients file if `.age-recipients` doesn't exist
- Add validation toggle in TUI editor (enable/disable format validation during editing)
- Use `age.GenerateX25519Identity()` directly instead of shelling out
- Maintain backward compatibility: all existing flags continue to work exactly as before
- Auto-detect non-interactive environments (CI/CD) and skip prompts

**Key Behaviors:**
- Prompts only appear when defaults are missing AND stdin is a TTY
- Use Bubble Tea components (textinput, list) for consistent UX
- Identity generation creates new key using the age library
- Validation toggle in editor (e.g., Ctrl+V) allows users to enable/disable format checking
- Always use ASCII armor output (no prompt needed)

## Impact
- **Affected specs**: `cli-wizard` (new capability)
- **Affected code**:
  - `cmd/agepad/main.go` - Add wizard orchestration before running editor
  - New `wizard/` package - Bubble Tea prompts for file/identity/recipients
  - `age/age.go` - Add identity generation and file writing functions
  - `tui/tui.go` - Add validation toggle state and keybinding
- **Breaking**: No - all existing flags and behavior remain unchanged
- **Dependencies**: Already have `charmbracelet/bubbles` and `filippo.io/age` - no new deps needed
