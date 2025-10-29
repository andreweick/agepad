# Build the agepad binary
build:
    mkdir -p bin
    go build -o bin/agepad ./cmd/agepad

# Run agepad with command line arguments
run *args: build
    ./bin/agepad {{args}}

# Run all tests
test:
    go test ./...

# Run tests with verbose output
test-verbose:
    go test -v ./...

# Clean build artifacts
clean:
    rm -rf bin

# Install agepad to GOPATH/bin
install:
    go install ./cmd/agepad
