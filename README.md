# Ink CLI

The command-line interface for [Ink](https://ml.ink) — a cloud platform designed for AI agents to deploy and manage services autonomously. Ink makes deployments simple enough that fully autonomous agents can handle the entire lifecycle: create, deploy, monitor, and scale services without human intervention.

## Install

**Homebrew:**

```bash
brew install mldotink/tap/ink
```

**npm:**

```bash
npm install -g @mldotink/ink-cli
```

**From source:**

```bash
go install github.com/mldotink/cli@latest
```

Or download a binary from [Releases](https://github.com/mldotink/cli/releases).

## Quick Start

```bash
# Authenticate
ink login

# Deploy a service
ink deploy my-app --repo my-repo --port 3000

# List services
ink list

# View service details
ink status my-app

# Tail logs
ink logs my-app

# Redeploy
ink redeploy my-app
```

## Configuration

Ink CLI resolves configuration in this order (highest priority first):

1. **CLI flags** — `--api-key`, `--workspace`, `--project`
2. **Environment** — `INK_API_KEY`
3. **Local config** — `.ink` file in current directory
4. **Global config** — `~/.config/ink/config`

Set workspace/project context:

```bash
# Per-project (creates .ink file, auto-added to .gitignore)
ink login --scope local

# Global default
ink login
```

## Commands

```
ink deploy <name>           Deploy a new service or update existing
ink redeploy <name>         Redeploy with optional config changes
ink list                    List all services (aliases: ls, services)
ink status <name>           Show service details
ink logs <name>             Tail service logs
ink delete <name>           Delete a service

ink db create <name>        Create a database
ink db list                 List databases
ink db delete <name>        Delete a database
ink db token <name>         Get database connection token

ink domains add <svc> <d>   Add custom domain
ink domains remove <svc> <d> Remove custom domain

ink dns zones               List DNS zones
ink dns records <zone>      List DNS records
ink dns add <zone>          Add DNS record
ink dns delete <zone> <id>  Delete DNS record

ink repos create <name>     Create internal git repo
ink repos token <name>      Get repo push token

ink projects list           List projects
ink projects delete <slug>  Delete a project

ink workspaces              List workspaces
ink workspaces create       Create workspace
ink workspaces members      List members
ink workspaces invite       Invite member

ink chat <message>          Ask the AI assistant
ink whoami                  Show current user
ink login                   Authenticate
```

## Flags

```
--json              Output as JSON
--workspace, -w     Workspace slug
--project           Project slug
--api-key           API key (overrides config)
```

## License

MIT
