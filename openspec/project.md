# Project Context

## Purpose
**agepad** is a secure terminal user interface (TUI) editor for editing AGE-encrypted files entirely in memory. It provides a user-friendly way to encrypt/decrypt and edit sensitive configuration files using the AGE encryption library without ever writing plaintext to disk.

Key features:
- All plaintext editing happens in RAM only (never touches disk)
- Prevents accidental exposure of secrets through temporary files
- Interactive TUI with diff previews, format validation, and recipient health checks
- Batch rotation of encrypted files with new recipients
- Environment variable injection from encrypted files

## Tech Stack
- **Go 1.25.1** - Primary language
- **Bubble Tea** (charmbracelet/bubbletea) - Terminal UI framework
- **urfave/cli v3** - Command-line argument parsing
- **filippo.io/age** - Modern, simple encryption library
- **go-toml/v2** - TOML validation
- **yaml.v3** - YAML validation
- **go-difflib** - Unified diff generation
- **charmbracelet/bubbles** - UI components (textarea)

## Project Conventions

### Code Style
- Variable naming: `req` for requests, `res` for responses
- SQL queries: prefer lowercase (when applicable)
- Testing: use subtests with descriptive names explaining what's being tested and expected result
- CLI: use double-dash flags (e.g., `--file`, `--recipients-file`)
- Single binary compilation - no shelling out to external commands
- Error handling: descriptive errors with context

### Architecture Patterns
Package structure follows typical Go application patterns:
- `cmd/agepad/` - Main entry point with CLI definition (uses urfave/cli)
- `model/` - Domain types (Config, RotateConfig, RunConfig)
- `age/` - AGE encryption/decryption operations
- `validator/` - Format validation (.env, JSON, YAML, TOML)
- `tui/` - Bubble Tea TUI editor state and logic

Design patterns:
- Atomic file writes using temp file + rename pattern
- Panic recovery with crash guards in main
- Two-step confirmation for destructive operations
- Health checks after encryption to prevent lock-out scenarios

### Testing Strategy
- Write tests for most functions and methods
- Use subtests with clear descriptions of behavior and expected results
- Prefer real implementations over mocks when testing core functionality
  - Example: tests use real AGE encryption with generated identities, not mocks
- Tests must not rely on execution order (they are shuffled)
- Each test should be independent and start with clean state

### Git Workflow
- Commit messages: surround identifier names (variable names, type names, etc.) in backticks
- Example: "Fix `DecryptToMemory` to handle armored format"
- Main branch: `main`

## Domain Context
**AGE Encryption:**
- AGE is a modern, simple file encryption tool/library
- Uses public key cryptography with recipients (public keys) and identities (private keys)
- Supports ASCII-armored output (base64-encoded, like PGP armor)
- Files typically use `.age` extension

**Security Model:**
- Plaintext only exists in memory during editing
- No temporary files or disk writes of unencrypted data
- Atomic writes prevent partial/corrupted encrypted states
- Recipient health checks: after encryption, immediately decrypt to verify access

**File Format Support:**
- Validates JSON, YAML, TOML, and `.env` formats before encryption
- `.env` format: `KEY=VALUE` pairs, one per line, with validation for proper key names

## Important Constraints
**Security-First Design:**
- Never write plaintext to disk (all editing in RAM)
- Prevent accidental data exposure through temporary files
- Atomic file operations to prevent corruption
- Health checks to prevent user lock-out

**User Experience:**
- Provide clear feedback on validation errors
- Warn on unsaved changes
- Two-step confirmation for saves (Ctrl+S twice)
- Diff preview before saving (Ctrl+D)

**Technical:**
- Single binary compilation
- Terminal-based UI (no GUI dependencies)
- Must work with standard AGE tooling

## External Dependencies
**Core Dependencies:**
- **filippo.io/age** (v1.2.1) - AGE encryption/decryption
- **github.com/charmbracelet/bubbletea** (v1.3.10) - TUI framework
- **github.com/charmbracelet/bubbles** (v0.21.0) - UI components
- **github.com/urfave/cli/v3** (v3.5.0) - CLI parsing
- **github.com/pelletier/go-toml/v2** (v2.2.4) - TOML validation
- **gopkg.in/yaml.v3** (v3.0.1) - YAML validation
- **github.com/pmezard/go-difflib** (v1.0.0) - Diff generation

**No External Services:**
- All operations are local
- No cloud services or APIs
- No databases
