# rosadisk-agent

API service for Rosadisk Agent, following OpenAPI 3 specification.

## Installation

### Debian/Ubuntu (Recommended)

Add the rosadisk-agent repository to your system:

```bash
# Import GPG key
wget -qO- https://rosavpn.github.io/rosadisk-agent/key.gpg | sudo gpg --dearmor -o /etc/apt/trusted.gpg.d/rosadisk-agent.gpg

# Add repository
echo "deb https://rosavpn.github.io/rosadisk-agent/ trixie main" | sudo tee /etc/apt/sources.list.d/rosadisk-agent.list
```

### From Source

See [Development](#development) section below.

## Getting Started

### Prerequisites

- Go 1.21+
- Make
- oapi-codegen: `go get github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen`
- oapi-codegen runtime: `go get github.com/oapi-codegen/runtime`
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
| `/v1/disks` | GET | List available disks |
| `/v1/fs` | GET, POST | List/create btrfs filesystems |
| `/v1/mounts` | GET, POST | List/mount btrfs filesystems |
| `/v1/subvolumes` | GET, POST | List/create subvolumes |
| `/v1/subvolumes/{id}` | GET, DELETE | Get/delete subvolume |

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
  database/
    database.go         # SQLite initialization and migrations
    subvolumes.go       # Subvolume SQL queries
  event/
    types.go            # Event-driven request/response types
  server/
    server.go           # Server implementation
    docs.html           # Swagger UI page
  storage/
    disks.go            # Disk listing operations
    filesystems.go      # Filesystem create/list operations
    mounts.go           # Mount operations
    subvolumes.go       # btrfs subvolume operations
Makefile                # generate, run, build targets
```

## Automatic Validation

The generated code uses `github.com/oapi-codegen/runtime` to provide automatic request validation:

- **Path parameters** — UUID format validation, type parsing (handled by `ServerInterfaceWrapper`)
- **Request bodies** — Type-safe JSON binding to generated structs via Echo's `ctx.Bind()`
- **Enum validation** — Generated enum types with `Valid()` method for runtime checks
- **OpenAPI spec** — Embedded and served at `/openapi.json` and `/openapi.yaml`

All API handlers implement the generated `ServerInterface`, ensuring compile-time type safety.
