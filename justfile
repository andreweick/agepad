# justfile for agepad - Secure AGE-encrypted file editor

# Default recipe - show available commands
default:
    @just --list

# Initialize Go module
init:
    go mod init github.com/andreweick/agepad

# Get dependencies
deps:
    go get filippo.io/age
    go get github.com/charmbracelet/bubbletea
    go get github.com/charmbracelet/bubbles
    go get github.com/spf13/pflag
    go get github.com/pmezard/go-difflib/difflib
    go get gopkg.in/yaml.v3
    go get github.com/pelletier/go-toml/v2
    go mod tidy

# Build the binary
build:
    go build -o agepad

# Build with optimizations (smaller binary)
build-release:
    go build -ldflags="-s -w" -o agepad

# Clean build artifacts
clean:
    rm -f agepad

# Full setup: init, deps, and build
all: init deps build

# Run tests
test:
    go test ./...

# Format code
fmt:
    go fmt ./...

# Run go vet
vet:
    go vet ./...

# Install binary to $GOPATH/bin
install:
    go install

# Check for common issues
check: fmt vet test

# Development build (with race detector)
build-dev:
    go build -race -o agepad
