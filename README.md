# rosadisk-agent

API service for Rosadisk Agent, following OpenAPI 3 specification.

## Getting Started

### Prerequisites

- Go 1.21+
- Make
- oapi-codegen: `go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`

### Build

```bash
make build
```

### Run

```bash
make run
```

The server starts on `:8080` by default.

## Endpoints

| Path | Method | Description |
|------|--------|-------------|
| `/_health` | GET | Health check |
| `/openapi.json` | GET | OpenAPI spec (JSON) |
| `/openapi.yaml` | GET | OpenAPI spec (YAML) |
| `/docs` | GET | Swagger UI |

## Development

### Regenerate code from OpenAPI spec

```bash
make generate
```

This runs `oapi-codegen` to regenerate types and server interface from `api/openapi.yaml`.

## CI Pipeline

This project uses GitHub Actions for continuous integration. The pipeline runs on every pull request and push to `main`.

### Pipeline Jobs

| Job | Description |
|-----|-------------|
| Lint | Runs golangci-lint for code quality checks |
| Test & Coverage | Runs unit tests with race detection and coverage threshold (50%) |
| Build | Builds the binary using `make build` |
| Security | Runs Gosec and Trivy for vulnerability scanning |
| Validate OpenAPI | Validates the OpenAPI specification |
| Docs | Verifies generated code is up to date |

## Project Structure

```
api/
  openapi.yaml          # OpenAPI 3 spec (source of truth)
  embed.go              # Embeds openapi.yaml for serving
  gen/
    server.gen.go       # Generated types & Echo router
cmd/
  server/
    main.go             # Entry point
internal/
  server/
    server.go           # Server implementation
    docs.html           # Swagger UI page
Makefile                # generate, run, build targets
```
