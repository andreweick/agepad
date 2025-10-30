# CLI Interactive Wizard Specification

This spec defines interactive prompting behavior for agepad when default configuration files are missing.

## ADDED Requirements

### Requirement: Interactive File Path Prompting
When the file path is not provided via flag and stdin is a TTY, the system SHALL prompt the user interactively using Bubble Tea.

#### Scenario: Missing file path in interactive session
- **WHEN** user runs `agepad` with no `--file` flag
- **AND** stdin is a TTY (interactive terminal)
- **THEN** agepad SHALL display a Bubble Tea textinput prompt: "Path to .age file to edit:"
- **AND** SHALL accept user input as the file path
- **AND** SHALL validate the path exists before proceeding

#### Scenario: Non-interactive session skips prompts
- **WHEN** user runs `agepad` with no `--file` flag
- **AND** stdin is not a TTY (piped/scripted)
- **THEN** agepad SHALL display an error: "Missing required flag: --file"
- **AND** SHALL exit with non-zero status

### Requirement: Interactive Identity File Prompting
When the default identity file doesn't exist, the system SHALL prompt with option to generate or specify an existing file.

#### Scenario: Missing identity file with generation option
- **WHEN** `~/.config/age/key.txt` does not exist
- **AND** stdin is a TTY
- **THEN** agepad SHALL display a Bubble Tea list prompt with description:
  - "AGE identity (private key) not found at ~/.config/age/key.txt"
  - "This is your private key used to decrypt files."
  - Options:
    - "Generate new identity at ~/.config/age/key.txt"
    - "Specify path to existing identity file"
- **AND** SHALL wait for user selection

#### Scenario: Generate new identity
- **WHEN** user selects "Generate new identity"
- **THEN** agepad SHALL call `age.GenerateX25519Identity()`
- **AND** SHALL create `~/.config/age` directory if it doesn't exist
- **AND** SHALL write identity to `~/.config/age/key.txt` with file permissions 0600
- **AND** SHALL display the generated public key to the user
- **AND** SHALL ask: "Add this public key to recipients file? [Y/n]"
- **AND** if confirmed, SHALL append public key to `.age-recipients`

#### Scenario: Specify existing identity path
- **WHEN** user selects "Specify path to existing identity file"
- **THEN** agepad SHALL display textinput prompt: "Path to identity file:"
- **AND** SHALL validate file exists and contains valid age identity
- **AND** SHALL re-prompt if validation fails

### Requirement: Interactive Recipients File Prompting
When the default recipients file doesn't exist, the system SHALL prompt for the recipients file path.

#### Scenario: Missing recipients file prompts with description
- **WHEN** `.age-recipients` file does not exist
- **AND** stdin is a TTY
- **THEN** agepad SHALL display textinput prompt with description:
  - "Recipients file not found. This file contains public keys for encryption."
  - "Path to recipients file (default: .age-recipients):"
- **AND** SHALL accept user input or use `.age-recipients` if Enter pressed
- **AND** SHALL validate file exists and contains at least one valid public key

#### Scenario: Offer to create missing recipients file
- **WHEN** user provides a recipients file path that doesn't exist
- **THEN** agepad SHALL prompt: "Recipients file doesn't exist. Create it now? [y/N]"
- **AND** if user confirms, SHALL create an empty file with helpful comment header
- **AND** SHALL prompt: "Add public key (one per line, empty to finish):"
- **AND** SHALL allow user to paste/type keys until empty line entered
- **AND** SHALL validate at least one key was added

### Requirement: TUI Validation Toggle
The TUI editor SHALL provide a toggle to enable/disable format validation during editing.

#### Scenario: Validation toggle keybinding
- **WHEN** user is in the TUI editor
- **AND** user presses Ctrl+V
- **THEN** agepad SHALL toggle validation on/off
- **AND** SHALL display validation status in the editor status bar

#### Scenario: Validation enabled shows format errors
- **WHEN** validation is enabled
- **AND** the file extension is .json, .yaml, .yml, .toml, or .env
- **THEN** agepad SHALL validate the current content using the appropriate validator
- **AND** SHALL display validation status: "✓ Valid JSON" or "✗ Invalid JSON: unexpected token at line 5"
- **AND** SHALL update validation on content change

#### Scenario: Validation disabled shows no errors
- **WHEN** validation is disabled
- **THEN** agepad SHALL display: "Validation: OFF"
- **AND** SHALL NOT run any format validation
- **AND** SHALL allow saving regardless of content validity

#### Scenario: Validation persists across editor session
- **WHEN** user toggles validation on
- **THEN** validation SHALL remain on until user toggles it off
- **AND** validation state SHALL NOT persist across different agepad invocations

### Requirement: Non-Interactive Mode Detection
The system SHALL automatically detect non-interactive environments and skip all prompts.

#### Scenario: Piped input skips prompts
- **WHEN** stdin is not a TTY (piped/redirected)
- **THEN** agepad SHALL NOT display any interactive prompts
- **AND** SHALL require all configuration via flags or default files
- **AND** SHALL fail with clear error messages if required configuration is missing

#### Scenario: CI/CD environment detection
- **WHEN** environment variable `CI=true` is set
- **THEN** agepad SHALL behave as non-interactive
- **AND** SHALL NOT prompt for any input
- **AND** SHALL fail immediately if defaults are missing

### Requirement: Graceful Exit from Wizard
The wizard SHALL allow users to exit at any point without completing the wizard.

#### Scenario: Ctrl+C exits wizard
- **WHEN** user presses Ctrl+C during any wizard prompt
- **THEN** agepad SHALL exit immediately with status 130 (typical SIGINT exit code)
- **AND** SHALL NOT create any partial files
- **AND** SHALL NOT leave the terminal in a bad state
