# build-cli

Command-line interface for the [BuilderHub](https://builder-hub.dev) platform.

The `builderhub` CLI is the one-stop shop for authenticating, managing organizations, builders, and API keys against the [build-api](https://github.com/builderhub/build-api) REST service.

## Install

```bash
go install github.com/builderhub/build-cli/cmd/builderhub@latest
```

Or build from source:

```bash
make build
./bin/builderhub version
```

## Quick start

```bash
# Configure your BuilderHub instance (API at https://api.<domain>)
builderhub config set domain builder-hub.dev

# Authenticate (stores JWT in ~/.config/builderhub/config.yaml)
builderhub auth login --email you@example.com

# Set default organization for builder commands
builderhub config set organization org_abc123

# Builder CRUD
builderhub builder list
builderhub builder create my-builder --mode sleepy --replicas 1 --label size=medium
builderhub builder get my-builder
builderhub builder update my-builder --mode persistent
builderhub builder wake my-builder
builderhub builder delete my-builder --yes
```

## Configuration

Config is stored at `$XDG_CONFIG_HOME/builderhub/config.yaml` (default: `~/.config/builderhub/config.yaml`).

The CLI targets `https://api.<domain>` for all API requests. Configure the instance domain once; there is no separate API URL setting.

| Setting | Config key | Environment variable |
|---------|------------|----------------------|
| Instance domain | `domain` | `BUILDERHUB_DOMAIN` |
| Bearer token | `api-key` or JWT via login | `BUILDERHUB_TOKEN` |
| Default organization | `organization` | — |

```bash
builderhub config set domain builder-hub.dev
builderhub config set organization org_abc123
builderhub config view
```

Global flags override config:

- `--domain` — BuilderHub instance domain (API at `https://api.<domain>`)
- `--profile` — named profile
- `-o, --organization` — default organization namespace
- `--token` — bearer token override (JWT or `bh_...` API key)
- `-O, --output` — `table` (default), `json`, or `yaml`

### Local development

Point at a local instance by setting the domain to `localhost` (requires build-api at `https://api.localhost`, e.g. via Tilt or ingress):

```bash
builderhub config set domain localhost
```

## Commands

### Auth

```bash
builderhub auth login [--email] [--password]
builderhub auth register --email ... --password ... --name ...
builderhub auth logout
builderhub auth whoami
builderhub auth refresh
```

### API keys

API key management requires a JWT session (run `auth login` first). API keys cannot create or revoke other API keys.

```bash
builderhub api-key list
builderhub api-key create ci-key --scope builders:read --scope builders:write
builderhub api-key delete <id> --yes
```

Valid scopes: `organizations:read`, `organizations:write`, `builders:read`, `builders:write`.

### Organizations

```bash
builderhub org list
builderhub org get <id>
builderhub org create --name "My Org" --slug my-org
builderhub org update <id> --name "Renamed"
builderhub org delete <id> --yes
builderhub org members list <org-id>
```

### Builders

Builder namespace is the organization ID.

```bash
builderhub builder list
builderhub builder get <name>
builderhub builder create <name> --mode sleepy|persistent|ephemeral [--replicas N] [--idle-timeout SEC] [--template-ref REF] [--label k=v]
builderhub builder update <name> [spec flags]
builderhub builder delete <name> [--yes]
builderhub builder wake <name>
```

### Other

```bash
builderhub health
builderhub version
builderhub completion bash
```

## Scripting with API keys

```bash
export BUILDERHUB_TOKEN=bh_...
export BUILDERHUB_DOMAIN=builder-hub.dev
builderhub -o org_abc123 -O json builder list
```

## Development

```bash
make test
make build
make install
```

With Nix:

```bash
nix develop
builderhub version
```

## License

MIT
