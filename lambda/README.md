# Lambda Package

This folder contains Lambda runtime code and packaging logic, isolated from infrastructure code.

## Purpose

- Keep Lambda implementation independent from deployment tooling.
- Make it easy to swap runtime/language later.

## Current structure

- Entrypoint: `cmd/handler/main.go`
- Handler/business logic: `internal/handler/handler.go`
- Go module: `go.mod`
- Build artifacts: `build/`

## Build

From repo root:

```bash
make package-lambda
```

Or directly from this folder:

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/bootstrap ./cmd/handler
zip -j build/lambda.zip build/bootstrap
```

Output artifact expected by Pulumi:

- `build/lambda.zip`
