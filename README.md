# rosadisk-agent

API service for Rosadisk Agent, following OpenAPI 3 specification.

## Installation

### Debian/Ubuntu (Recommended)

Add the rosadisk-agent repository to your system:

```bash
# Import GPG key
wget -qO- https://rosadisk.github.io/rosadisk-agent/key.gpg | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/rosadisk-agent.gpg

# Add repository
echo "deb https://rosadisk.github.io/rosadisk-agent/ trixie main" | sudo tee /etc/apt/sources.list.d/rosadisk-agent.list

# Install
sudo apt update
sudo apt install rosadisk-agent
```

### From Source

See [Development](#development) section below.

## Getting Started

### Prerequisites

- Go 1.21+
- Make
- oapi-codegen: `go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
- pre-commit: `pip install pre-commit`

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

### Install pre-commit hooks

```bash
pre-commit install
```

Hooks run automatically on `git commit` to enforce formatting, linting, spec validation, and generated code consistency.

### Regenerate code from OpenAPI spec

```bash
make generate
```

This runs `oapi-codegen` to regenerate types and server interface from `api/openapi.yaml`.

### Build Debian Repository (Local Testing)

```bash
make deb-repo
```

This builds a local Debian repository structure for testing. Requires `dpkg-dev` and GPG key setup.

### GitHub Secrets for Debian Repository

The following secrets must be configured in GitHub for the Debian repository workflow:

| Secret | Description |
|--------|-------------|
| `DEB_GPG_KEY` | GPG private key (ASCII armored) for signing the repository |
| `DEB_GPG_KEY_ID` | GPG key ID (fingerprint) |

To export your GPG key for the secret:
```bash
gpg --armor --export-secret-key rosa-agent@noreply.github.com
```

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
