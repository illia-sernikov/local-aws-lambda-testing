# Lambda + API Gateway on LocalStack (Go)

This repository is a local-first sandbox for AWS Lambda development with LocalStack Community Edition.
It packages a Go Lambda, deploys it behind API Gateway with Pulumi, and provides quick ways to invoke and verify it without using real AWS resources.

## API

- `GET /healthcheck`
- `POST /calculate`

`POST /calculate` expects:

```json
{
  "a": 10,
  "b": 2,
  "operation": "divide"
}
```

Supported operations: `add`, `subtract`, `multiply`, `divide`.

## Project layout

- Lambda entrypoint: `cmd/handler/main.go`
- Request handling and business logic: `internal/handler/handler.go`
- API contract used by API Gateway: `docs/openapi.yaml`
- Pulumi stack (LocalStack provider/resources): `infra/main.go`

## Prerequisites

- Go `1.24.13` (see `go.mod` / `mise.toml`)
- Docker and Docker Compose
- Pulumi CLI
- `zip`
- Optional: `awslocal` (for `make logs`)

LocalStack credentials are set to `test` in the stack and compose setup.

## Start LocalStack

```bash
docker compose up -d
```

This starts:

- `localstack` on `http://localhost:4566`
- `traefik` on `http://localhost` (dashboard on `http://localhost:8080`)
- `traefik-config-generator` to refresh Traefik routes from discovered API Gateway APIs

## Build

```bash
go mod tidy
go build ./...
make package-lambda
```

Build output:

- Linux Lambda bootstrap binary: `build/bootstrap`
- Lambda artifact zip used by Pulumi: `build/lambda.zip`

## Deploy (Pulumi to LocalStack)

```bash
cd infra
pulumi login --local
pulumi stack init local   # run once
pulumi up
```

You can also use the root shortcut:

```bash
make deploy
```

Useful stack config:

- `artifactPath` (default: `../build/lambda.zip`)
- `enableExecutionLogging` (default: `false`)

Example:

```bash
cd infra
pulumi config set artifactPath ../build/lambda.zip
pulumi config set enableExecutionLogging true
```

## Get endpoint and test

Get deployed API base URL from stack output:

```bash
API_URL=$(cd infra && pulumi stack output apiEndpoint)
```

Healthcheck:

```bash
curl "${API_URL}/healthcheck"
```

Calculate:

```bash
curl -X POST "${API_URL}/calculate" \
  -H "Content-Type: application/json" \
  -d '{"a":10,"b":2,"operation":"divide"}'
```

## Bruno collection

A ready-to-run Bruno collection is included in `bruno/localstack-lambda`.

- Collection file: `bruno/localstack-lambda/bruno.json`
- Environment: `bruno/localstack-lambda/environments/local.bru`
- Requests:
  - `bruno/localstack-lambda/healthcheck.bru`
  - `bruno/localstack-lambda/calculate-add.bru`

The local environment uses:

- `traefikUrl = http://localhost`
- `apiHost = lambda-api.api.localhost.localstack`

Requests are sent to Traefik and routed by the `Host` header, so make sure `docker compose up -d` is running before executing the collection.

## Logs and cleanup

Tail Lambda logs:

```bash
make logs
```

Destroy stack:

```bash
make destroy
```

Stop containers:

```bash
docker compose down
```
