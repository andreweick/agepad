# agepad

Securely edit AGE-encrypted files entirely in memory using a Bubble Tea TUI.

## Features

- **In-memory editing**: Plaintext never touches disk; editing happens in RAM via Bubble Tea textarea
- **ASCII-armored output**: Default armored output (disable with `--armor=false`)
- **Default identities**: Uses `~/.config/age/key.txt` with friendly guidance if missing
- **Diff-before-save**: Preview changes with Ctrl+D; confirm with double Ctrl+S
- **Format validation**: Syntax checks for `.env`, `.json`, `.yaml/.yml`, `.toml` before encrypting
- **Read-only mode**: View-only mode with `--view` flag
- **Recipient health check**: Preflight encryption/decryption test to prevent lock-out
- **Batch rotate**: Re-encrypt `*.age` files in a directory tree with new recipients
- **Env injection**: Export decrypted KEY=VALUE pairs into child process environment
- **Crash guard**: Helpful recovery messages (edits were only in RAM)

## Installation

```bash
go install github.com/andreweick/agepad/cmd/agepad@latest
```

Or build from source:

```bash
git clone https://github.com/andreweick/agepad
cd agepad
just build
```

## Usage

### Interactive TUI Editor

Edit an encrypted file:

```bash
agepad --file secrets/app.env.age --recipients-file .age-recipients
```

View-only mode:

```bash
agepad --file secrets/app.env.age --recipients-file .age-recipients --view
```

### Rotate Recipients

Re-encrypt all `.age` files in a directory tree with a new recipients set:

```bash
agepad rotate --root secrets --from .age-recipients --to .age-recipients.new --identities ~/.config/age/key.txt
```

### Environment Injection

Decrypt a file and inject its KEY=VALUE pairs into a child process environment:

```bash
agepad run -- secrets/app.env.age -- myserver --port 8080
```

This decrypts `secrets/app.env.age` and exports its variables to `myserver`, without creating temporary files.

## Keyboard Shortcuts (TUI Mode)

- **Ctrl+D**: Preview diff of changes
- **Ctrl+S**: Save (press twice to confirm if content changed)
- **Ctrl+Q**: Quit (press twice if there are unsaved changes)
- **Esc**: Alternative quit

## Configuration

### Recipients File

Create a `.age-recipients` file in your project root (recommended for repo commits):

```
age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p
age1lggyhqrw2nlhcxprm67z43rta597azn8gknawjehu9d9dl0jq3yqqvfafg
```

### Identity File

Generate an AGE identity if you don't have one:

```bash
age-keygen --output ~/.config/age/key.txt
```

Or specify a different path:

```bash
agepad --file secrets.age --identities /path/to/key.txt
```

## Project Structure

```
agepad/
├── cmd/
│   └── agepad/       # Main entry point with CLI
├── model/            # Domain types and configuration
├── age/              # AGE encryption/decryption operations
├── validator/        # Format validation for .env, JSON, YAML, TOML
├── tui/              # Bubble Tea TUI editor logic
├── go.mod
├── README.md
└── justfile          # Build automation
```

## Development

### Build

```bash
just build
```

### Run

```bash
just run --file test.age --recipients-file .age-recipients
```

### Test

```bash
go test ./...
```

## Security Notes

- Plaintext is only ever in RAM during editing sessions
- No temporary files are created
- Recipient health checks prevent lock-out scenarios
- Format validation prevents saving invalid configurations

## License

MIT

## Credits

Built with:
- [age](https://github.com/FiloSottile/age) - Simple, secure encryption
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - Terminal UI framework
- [urfave/cli](https://github.com/urfave/cli) - Command-line interface framework
