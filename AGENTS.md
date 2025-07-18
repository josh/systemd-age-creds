# Agents Guide

This project is written in Go and has no go mod dependencies. The repository assumes Go 1.24 or newer.

## Environment Setup

1. Run on X86 or ARM Linux with systemd
2. Install Go 1.24 or later.

## Testing

Run native go tests with:

```sh
go test ./...
```

Run tests inside Nix sandbox:

```
nix flake check
```

## Formatting

Format code with:

```sh
go fmt main.go main_test.go
```

## Code Quality

Run vet and static analysis tools before committing:

```sh
go vet main.go main_test.go
```

Optionally run `golangci-lint` for additional checks:

```sh
golangci-lint run main.go main_test.go
```

## Building

Build the project with:

```sh
go build main.go
```

## Comments

Keep comments concise. Only add them when they clarify non-obvious logic.
