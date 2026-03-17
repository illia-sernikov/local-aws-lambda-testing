# Simple AWS Lambda in Go

This project has one Lambda function with 2 API endpoints:

- `GET /healthcheck`
- `POST /calculate`

Binary entrypoint: `cmd/handler/main.go`.
Business logic is in `internal/handler`.

`calculate` expects JSON body:

```json
{
  "a": 10,
  "b": 2,
  "operation": "divide"
}
```

Supported operations: `add`, `subtract`, `multiply`, `divide`.

## Requirements

- Go 1.22+
- Pulumi CLI
- AWS credentials configured

## Build

```bash
go mod tidy
go build ./...
make package-lambda
```

## Deploy

```bash
cd infra
go mod tidy
pulumi stack init dev
pulumi up
```

By default, Pulumi expects lambda artifact at `../build/lambda.zip`.
You can override with config:

```bash
pulumi config set artifactPath ../build/lambda.zip
```

Pulumi also exports a static local base URL via custom domain.
Use `apiCustomBaseUrl` as `API_URL` so tests keep working after destroy/deploy.

Note: in LocalStack, `aws:apigateway:DomainName` can be slow or hang depending on edition/features.
Custom domain is disabled by default in this stack.

```bash
API_URL=$(cd infra && pulumi stack output apiCustomBaseUrl)
```

## Test

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

Default static local domain is `api.localhost.localstack.cloud` on port `4566`.
You can override it with:

```bash
cd infra
pulumi config set enableCustomDomain true
pulumi config set apiDomainName my-api.localhost.localstack.cloud
```
