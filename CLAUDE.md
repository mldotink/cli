# Ink CLI

Go CLI for the Ink PaaS (ml.ink). Built with Cobra + Charm lipgloss for terminal styling.

## Architecture

- `main.go` — entry point, injects version via ldflags (`-X main.version=...`)
- `cmd/` — one file per command, flat structure (no noun groups for services)
- `internal/api/` — GraphQL client with Bearer auth via `authTransport` RoundTripper
- `internal/config/` — config resolution: flags > env > `.ink` (local) > `~/.config/ink/config` (global)
- `internal/gql/` — genqlient-generated typed GraphQL client code
- `schema.graphql` — concatenated from backend `go-backend/internal/graph/*.graphqls` (server directives stripped)
- `npm/` — per-platform npm packages for `npx` distribution

## Code Generation

GraphQL types are generated with [genqlient](https://github.com/Khan/genqlient):

```bash
go run github.com/Khan/genqlient@main
```

Config: `genqlient.yaml`. Operations: `internal/gql/operations.graphql`. Output: `internal/gql/generated.go`.

Key config: `optional: pointer` — nullable GraphQL params become `*string` (backend distinguishes `null` from `""`).

## Schema Updates

The schema is sourced from backend `.graphqls` files (NOT API introspection, which may lag behind undeployed changes):

```bash
cat ../go-backend/internal/graph/*.graphqls | python3 -c "
import re, sys
s = sys.stdin.read()
s = re.sub(r'directive @\w+.*?\n', '', s)
s = re.sub(r'\s*@(goField|isAuthenticated|agent|hasRole)\b(\([^)]*\))?', '', s)
print(s)
" > schema.graphql
```

Then regenerate: `go run github.com/Khan/genqlient@main`

## Commands

| Command | File | Description |
|---------|------|-------------|
| `list` (aliases: `ls`, `services`) | services.go | List services with project mapping |
| `deploy` | deploy.go | Create or update+redeploy a service |
| `redeploy` | deploy.go | Redeploy existing service with optional config changes |
| `delete` | delete.go | Delete a service |
| `status` | status.go | Show service details |
| `logs` | logs.go | Tail service logs |
| `db create/list/delete/token` | databases.go | Manage databases |
| `domains add/remove` | domains.go | Custom domains |
| `dns zones/records/add/delete` | dns.go | DNS management |
| `repos create/token` | repos.go | Internal git repos |
| `projects list/delete` | projects.go | Project management |
| `workspaces` (+ members/invite) | workspaces.go | Workspace management |
| `chat` | chat.go | AI assistant |
| `login` | login.go | Authenticate |
| `whoami` | whoami.go | Show current user |

## Conventions

- All commands show workspace/project context via `printConfigHints()` (except login, help, completion, workspaces, whoami)
- `--json` flag on all commands for machine-readable output
- `newClient()` creates a `graphql.Client`; helpers `wsPtr()`, `projPtr()`, `ptr()` for nullable params
- `findService(name)` resolves a service by name within current workspace scope
- List command uses dual-query pattern (`serviceList` + `projectList`) to avoid N+1

## Release

Tag `v*` triggers `.github/workflows/release.yml`:
1. GoReleaser cross-compiles + creates GitHub Release + pushes Homebrew formula to `mldotink/homebrew-tap`
2. npm job downloads assets, runs `npm/publish.sh` to publish `@mldotink/ink-cli-{platform}` + meta package

## Build

```bash
go build -o ink .
```
