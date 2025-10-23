# agepad

Securely edit AGE-encrypted files entirely in memory with a beautiful Terminal UI (TUI).

## Overview

`agepad` is a security-focused editor for [AGE](https://age-encryption.org/)-encrypted files that keeps your secrets safe by never writing plaintext to disk. All editing happens in memory using the [Bubble Tea](https://github.com/charmbracelet/bubbletea) framework.

## Key Features

- **Memory-only editing**: Plaintext never touches disk - all editing happens in-process RAM
- **ASCII-armored output**: Default ASCII-armored `.age` files (disable with `--armor=false`)
- **Smart defaults**: Uses `~/.config/age/key.txt` by default with helpful setup guidance
- **Diff-before-save**: Preview changes with `Ctrl+D` before writing
- **Double-save confirmation**: `Ctrl+S` twice to prevent accidental overwrites
- **Format validation**: Syntax checks for `.env`, `.json`, `.yaml/.yml`, `.toml` files before encrypting
- **Read-only mode**: View secrets without editing (`--view` flag)
- **Recipient health checks**: Preflight encryption/decryption test to prevent lock-out
- **Batch recipient rotation**: Re-encrypt multiple `.age` files with new recipients
- **Environment injection**: Run commands with decrypted env vars without temp files
- **Crash guard**: Helpful recovery messages (buffer was only in RAM)

## Installation

### Using just (recommended)

```bash
just all
```

This will:
1. Initialize the Go module
2. Fetch all dependencies
3. Build the binary

### Manual build

```bash
# Initialize module
go mod init github.com/andreweick/agepad

# Get dependencies
go get filippo.io/age
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/spf13/pflag
go get github.com/pmezard/go-difflib/difflib
go get gopkg.in/yaml.v3
go get github.com/pelletier/go-toml/v2
go mod tidy

# Build
go build -o agepad
```

### Install to PATH

```bash
just install
# or
go install
```

## Prerequisites

You'll need an AGE key pair. If you don't have one:

```bash
mkdir -p ~/.config/age
age-keygen --output ~/.config/age/key.txt
```

Extract your public key for the recipients file:

```bash
age-keygen -y ~/.config/age/key.txt > .age-recipients
```

## Usage

### Interactive TUI Editor

Edit an encrypted file:

```bash
./agepad --file secrets/app.env.age --recipients-file .age-recipients
```

View-only mode (no editing):

```bash
./agepad --file secrets/app.env.age --recipients-file .age-recipients --view
```

#### TUI Keyboard Shortcuts

- **Ctrl+D**: Show diff preview (compare original vs. edited)
- **Ctrl+S**: Save (press twice to confirm changes)
- **Ctrl+Q**: Quit (press twice if unsaved changes)
- **Esc**: Quit

### Batch Recipient Rotation

Re-encrypt all `.age` files in a directory tree with new recipients:

```bash
./agepad rotate \
  --root secrets \
  --from .age-recipients \
  --to .age-recipients.new \
  --identities ~/.config/age/key.txt
```

This decrypts each file with your current identity and re-encrypts with the new recipients.

### Environment Injection

Run a command with environment variables from an encrypted file (no temp files created):

```bash
./agepad run -- secrets/app.env.age -- myserver --port 8080
```

The decrypted KEY=VALUE pairs are injected directly into the command's environment.

## Command-Line Options

### Main TUI Mode

```
--file string              Path to the .age file to edit (required)
--recipients-file string   Path to recipients file (default: .age-recipients)
--identities string        Path to AGE identities (default: ~/.config/age/key.txt)
--armor                    Write ASCII-armored .age output (default: true)
--view                     Open in read-only view mode
```

### Rotate Subcommand

```
--root string         Root directory to scan for *.age files (default: ".")
--from string         Current recipients file (for documentation)
--to string           NEW recipients file to use (required)
--identities string   AGE identities to decrypt during rotation
```

### Run Subcommand

```
agepad run -- <file.age> -- <command> [args...]
```

## Workflow Examples

### Initial Setup

```bash
# 1. Generate your AGE key
age-keygen --output ~/.config/age/key.txt

# 2. Create recipients file (add team members' public keys)
age-keygen -y ~/.config/age/key.txt > .age-recipients

# 3. Create your first encrypted file
echo "API_KEY=secret123" | age -r $(cat .age-recipients) -a -o secrets/app.env.age

# 4. Edit it securely
./agepad --file secrets/app.env.age
```

### Team Collaboration

```bash
# Share your public key with the team
age-keygen -y ~/.config/age/key.txt

# Add team members' public keys to .age-recipients (one per line)
cat .age-recipients
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
age1cy0su9fwf3gf9mw868g5yut09p6nytfmmnktexz2sg7tqm8h2jnqxjqgkt

# Commit .age-recipients to your repo
git add .age-recipients
git commit -m "Add team AGE recipients"

# Now anyone with their private key can edit
./agepad --file secrets/app.env.age
```

### Key Rotation

```bash
# Generate new key
age-keygen --output ~/.config/age/key-new.txt

# Create new recipients file
age-keygen -y ~/.config/age/key-new.txt > .age-recipients.new

# Rotate all secrets
./agepad rotate --root secrets --to .age-recipients.new

# Move new key to default location
mv ~/.config/age/key-new.txt ~/.config/age/key.txt
mv .age-recipients.new .age-recipients
```

### CI/CD Integration

```bash
# In CI, inject secrets without temp files
./agepad run -- config/prod.env.age -- ./deploy.sh

# Or decrypt for inspection (read-only)
./agepad --file config/prod.env.age --view
```

## File Format Support

agepad automatically validates these formats before encrypting:

- **`.env`**: KEY=VALUE pairs
- **`.json`**: Valid JSON syntax
- **`.yaml`**, **`.yml`**: Valid YAML syntax
- **`.toml`**: Valid TOML syntax
- **Other files**: No validation (raw text)

## Security Features

1. **No disk writes**: Plaintext only exists in process memory
2. **Atomic writes**: Temp file + rename to prevent corruption
3. **Preflight checks**: Test decrypt before writing to prevent lock-out
4. **Crash guard**: Helpful recovery messages (no data on disk to recover)
5. **Format validation**: Catch syntax errors before encrypting

## Design Philosophy

- **Long options only**: Explicit, readable commands
- **Fail fast**: Validate early, provide helpful error messages
- **Repo-friendly**: Commit `.age-recipients` for team collaboration and CI
- **Memory safety**: Never write plaintext to disk
- **User-friendly**: Clear prompts and guided setup

## Building from Source

### Standard build

```bash
just build
```

### Optimized release build

```bash
just build-release
```

### Development build (with race detector)

```bash
just build-dev
```

### Run tests and checks

```bash
just check
```

## Dependencies

- [filippo.io/age](https://pkg.go.dev/filippo.io/age) - AGE encryption
- [github.com/charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [github.com/charmbracelet/bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [github.com/spf13/pflag](https://github.com/spf13/pflag) - POSIX-style flags
- [github.com/pmezard/go-difflib](https://github.com/pmezard/go-difflib) - Diff generation
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML parsing
- [github.com/pelletier/go-toml/v2](https://github.com/pelletier/go-toml) - TOML parsing

## License

See LICENSE file for details.

## Contributing

Contributions welcome! Please ensure:
- Code passes `just check` (fmt, vet, test)
- Security-focused changes are well-documented
- New features maintain the "never write plaintext to disk" principle

## Troubleshooting

### "AGE private key not found"

```bash
age-keygen --output ~/.config/age/key.txt
```

### "Recipients file not found"

```bash
age-keygen -y ~/.config/age/key.txt > .age-recipients
```

### "Preflight decrypt failed"

Your current identity cannot decrypt with the specified recipients. Make sure your public key is in the recipients file:

```bash
age-keygen -y ~/.config/age/key.txt >> .age-recipients
```

### Locked out of encrypted file

If you lost your private key, the file cannot be recovered (that's the point of encryption!). Always:
- Back up your `~/.config/age/key.txt` securely
- Keep multiple team members in `.age-recipients`
- Test decryption after rotating keys
