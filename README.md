# @toppy/hookdeck-deploy-cli

Deploy Hookdeck webhook infrastructure from declarative manifest files.

Define your sources, destinations, connections, and transformations in `hookdeck.jsonc` files, then deploy them with a single command. Supports per-resource environment overrides, project-wide auto-discovery, drift detection, and variable interpolation.

## Install

```bash
npm install -D @toppy/hookdeck-deploy-cli
```

or

```bash
pnpm add -D @toppy/hookdeck-deploy-cli
```

The package includes prebuilt binaries for Linux, macOS, and Windows (amd64/arm64). No build tools required.

## Quick Start

**1. Create a manifest file** (`hookdeck.jsonc`):

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "sources": [
    { "name": "my-webhook-source" }
  ],
  "destinations": [
    {
      "name": "my-service",
      "url": "https://my-service.example.com/webhooks"
    }
  ],
  "connections": [
    {
      "name": "webhook-to-service",
      "source": "my-webhook-source",
      "destination": "my-service"
    }
  ]
}
```

**2. Preview changes** with a dry run:

```bash
hookdeck-deploy deploy --dry-run
```

**3. Deploy:**

```bash
hookdeck-deploy deploy
```

Resources are deployed in dependency order: source -> transformation -> destination -> connection.

For a complete multi-environment setup, see the [`example/`](./example) directory.

## Authentication

### Profiles (recommended)

Authenticate using the [Hookdeck CLI](https://hookdeck.com/docs/cli):

```bash
hookdeck login
```

This creates a config file at `~/.config/hookdeck/config.toml` with your API key. For multi-environment setups, add named profiles:

```toml
profile = "default"

[default]
api_key = "hk_..."

[staging]
api_key = "hk_..."
project_id = "prj_..."

[production]
api_key = "hk_..."
project_id = "prj_..."
```

Reference profiles in your project configuration (see [Project Mode](#project-mode)):

```jsonc
// hookdeck.project.jsonc
{
  "version": "2",
  "env": {
    "staging": { "profile": "staging" },
    "production": { "profile": "production" }
  }
}
```

### Environment variable

For CI/CD pipelines, set `HOOKDECK_API_KEY`. This takes priority over profile-based credentials:

```bash
HOOKDECK_API_KEY=hk_... hookdeck-deploy deploy --env production
```

### Resolution order

1. `HOOKDECK_API_KEY` environment variable
2. Named profile from project config's `env.<name>.profile`
3. Default profile from config file

Config file locations (checked in order):
- `.hookdeck/config.toml` (project-local)
- `~/.config/hookdeck/config.toml` (global)

## Manifest Guide

### Sources

Define Hookdeck sources to receive webhooks:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "sources": [
    {
      "name": "order-webhook",
      "description": "Receives order webhooks from the payment provider"
    }
  ]
}
```

Sources support per-environment overrides for `type`, `description`, and `config`:

```jsonc
"sources": [
  {
    "name": "order-webhook",
    "env": {
      "production": {
        "type": "Stripe",
        "config": { "webhook_secret_key": "${STRIPE_WEBHOOK_SECRET}" }
      }
    }
  }
]
```

After deploying, the source URL from Hookdeck is automatically synced back to your `wrangler.jsonc` (disable with `--sync-wrangler=false`).

### Destinations

Define where events are delivered:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "destinations": [
    {
      "name": "order-processor",
      "url": "https://order-processor-dev.example.com/webhooks",
      "rate_limit": 10,
      "rate_limit_period": "second",
      "env": {
        "staging": {
          "url": "https://order-processor-staging.example.com/webhooks"
        },
        "production": {
          "url": "https://order-processor.example.com/webhooks",
          "rate_limit": 50
        }
      }
    }
  ]
}
```

Destination overrides support: `url`, `type`, `description`, `auth_type`, `auth`, `config`, `rate_limit`, and `rate_limit_period`.

### Connections

Wire a source to a destination with optional filtering and transformations:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "connections": [
    {
      "name": "orders-to-processor",
      "source": "order-webhook",
      "destination": "order-processor",
      "transformations": ["enrich-order"],
      "filter": {
        "$or": [
          { "headers.x-event-type": "order.created" }
        ]
      }
    }
  ]
}
```

Connections reference sources and destinations by name. Both `filter` and `transformations` are shorthands that get converted to rules during deployment.

Filters use a MongoDB-like query syntax with operators like `$and`, `$or`, and `$exist`:

```jsonc
"filter": {
  "$or": [
    { "headers.x-event-type": "order.created" },
    {
      "$and": [
        { "headers.x-event-type": "order.updated" },
        { "body.status": { "$exist": true } }
      ]
    }
  ]
}
```

Connections support per-environment overrides for `filter`, `transformations`, `rules`, `source`, and `destination`:

```jsonc
"connections": [
  {
    "name": "orders-to-processor",
    "source": "order-webhook",
    "destination": "order-processor",
    "transformations": ["enrich-order"],
    "filter": { "headers.x-event-type": "order.created" },
    "env": {
      "staging": {
        "filter": {
          "$or": [
            { "headers.x-event-type": "order.created" },
            { "headers.x-event-type": "order.updated" }
          ]
        }
      },
      "production": {
        "transformations": []
      }
    }
  }
]
```

### Transformations

Define transformations with a JavaScript source file. The `code_file` path is resolved relative to the manifest file:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "transformations": [
    {
      "name": "enrich-order",
      "description": "Adds computed fields to order payload",
      "code_file": "handler.js",
      "env": {
        "API_BASE_URL": "https://api-dev.example.com"
      },
      "env_overrides": {
        "staging": {
          "env": { "API_BASE_URL": "https://api-staging.example.com" }
        },
        "production": {
          "env": { "API_BASE_URL": "https://api.example.com" }
        }
      }
    }
  ]
}
```

The JavaScript file must use the Hookdeck transformation signature:

```js
addHandler("transform", (request, context) => {
  // Access environment variables defined in the manifest
  const apiUrl = context.env.API_BASE_URL;

  // Modify the request before delivery
  request.body.processed_at = new Date().toISOString();
  return request;
});
```

Environment variables defined in `env` are available at runtime via `context.env`. Use `env_overrides` to set different values per environment.

> **Note:** Transformations use `env_overrides` (not `env`) for per-environment overrides, because `env` already holds the runtime environment variables passed to the transformation code.

### Environment Overrides

Deploy to specific environments with `--env`:

```bash
hookdeck-deploy deploy --env staging
hookdeck-deploy deploy --env production
```

Each resource defines its own `env` map for per-environment overrides. Only the fields you specify are overridden — everything else uses the base values:

```jsonc
"destinations": [
  {
    "name": "my-service",
    "url": "https://my-service-dev.example.com",
    "rate_limit": 10,
    "rate_limit_period": "second",
    "env": {
      "production": {
        "url": "https://my-service.example.com",
        "rate_limit": 100
      }
    }
  }
]
```

Without `--env`, the base values are used (useful for local development).

### Variable Interpolation

Reference environment variables in manifest values with `${VAR_NAME}`:

```jsonc
{
  "destinations": [
    {
      "name": "my-service",
      "auth_type": "HOOKDECK_SIGNATURE",
      "auth": {
        "webhook_secret_key": "${HOOKDECK_SIGNING_SECRET}"
      }
    }
  ]
}
```

Variables are resolved from the process environment at deploy time.

## Project Mode

For repositories with multiple webhook integrations, use **project mode** to deploy all manifests at once.

**1. Create a project config** (`hookdeck.project.jsonc`) in your project root:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-project.schema.json",
  "version": "2",
  "env": {
    "staging": { "profile": "staging" },
    "production": { "profile": "production" }
  }
}
```

**2. Organize manifests** in subdirectories:

```
hookdeck.project.jsonc         # Project config: environment profiles
sources/
  order-webhook/
    hookdeck.jsonc              # Source definition
destinations/
  order-processor/
    hookdeck.jsonc              # Destination definition
transformations/
  enrich-order/
    hookdeck.jsonc              # Transformation manifest
    handler.js                  # Transformation code
connections/
  orders-to-processor/
    hookdeck.jsonc              # Connection definition
```

**3. Deploy everything:**

```bash
hookdeck-deploy deploy --env staging
```

When `hookdeck.project.jsonc` exists in the working directory, project mode activates automatically. All `hookdeck.jsonc` files under the project root are discovered and deployed in dependency order.

You can also point to a project config explicitly:

```bash
hookdeck-deploy deploy --project path/to/hookdeck.project.jsonc --env production
```

See the [`example/`](./example) directory for a working project-mode layout.

### Deploy Scripts

A typical `package.json` setup:

```json
{
  "scripts": {
    "deploy:staging": "hookdeck-deploy deploy --env staging",
    "deploy:production": "hookdeck-deploy deploy --env production",
    "drift:staging": "hookdeck-deploy drift --env staging"
  }
}
```

## CLI Reference

### Commands

| Command | Description |
|---------|-------------|
| `hookdeck-deploy deploy` | Upsert resources in dependency order (source -> transformation -> destination -> connection) |
| `hookdeck-deploy drift` | Compare manifest against live Hookdeck state, report missing or drifted resources |
| `hookdeck-deploy status` | Show whether each manifest resource exists on Hookdeck with name, ID, and URL |
| `hookdeck-deploy schema` | Output JSON schema for manifest files |

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--file <path>` | `-f` | Manifest file path (default: `hookdeck.jsonc` or `hookdeck.json`) |
| `--env <name>` | `-e` | Environment overlay (e.g., `staging`, `production`) |
| `--dry-run` | | Preview changes without applying |
| `--profile <name>` | | Override credential profile |
| `--project <path>` | | Path to `hookdeck.project.jsonc` for project-wide deploy |

### Deploy Flags

| Flag | Description |
|------|-------------|
| `--sync-wrangler` | Sync source URL back to `wrangler.jsonc` after deploy (default: `true`) |

### Schema Flags

| Flag | Description |
|------|-------------|
| `--project` | Output the project configuration schema instead of the deploy schema |

## JSON Schemas

Add a `$schema` property to your manifests for IDE autocompletion and validation:

```jsonc
// Resource manifests (sources, destinations, connections, transformations):
{ "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json" }

// Project configuration:
{ "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-project.schema.json" }
```

## Contributing

### Prerequisites

- [Go 1.24+](https://go.dev/dl/)

### Getting Started

```bash
git clone https://github.com/toppynl/hookdeck-deploy-cli.git
cd hookdeck-deploy-cli
go build -o hookdeck-deploy-cli .
go test ./...
```

### Build

```bash
go build -o hookdeck-deploy-cli .
```

### Test

```bash
go test ./...
```

## License

MIT
