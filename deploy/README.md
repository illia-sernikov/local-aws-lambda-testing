# Deploy Package

This folder contains infrastructure/deployment assets only.

## Layout

- Pulumi project: `pulumi/`
- Pulumi program: `pulumi/main.go`
- Traefik dynamic config generator: `traefik-config-generator/`
- Traefik dynamic config mount: `traefik/dynamic/`

The generator writes `traefik/dynamic/apis.yaml` at runtime.
It ensures the target directory exists before writing.

## Deploy with Pulumi (LocalStack)

```bash
cd deploy/pulumi
pulumi login --local
pulumi stack init local   # run once
pulumi up
```

Useful config:

- `artifactPath` (default: `../../lambda/build/lambda.zip`)
- `enableExecutionLogging` (default: `false`)

Example:

```bash
cd deploy/pulumi
pulumi config set artifactPath ../../lambda/build/lambda.zip
pulumi config set enableExecutionLogging true
```
