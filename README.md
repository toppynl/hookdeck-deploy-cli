# @toppy/hookdeck-deploy-cli

Deploy Hookdeck webhook infrastructure from declarative manifest files.

Define your sources, destinations, connections, and transformations in `hookdeck.jsonc` files, then deploy them with a single command. Supports environment overlays, manifest inheritance, drift detection, and variable interpolation.

## Install

```bash
npm install -D @toppy/hookdeck-deploy-cli
```

or

```bash
pnpm add -D @toppy/hookdeck-deploy-cli
```

The package includes prebuilt binaries for Linux, macOS, and Windows (amd64/arm64). No build tools required.

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

Reference profiles in your manifest:

```jsonc
{
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
2. Named profile from manifest's `env.<name>.profile`
3. Default profile from config file

Config file locations (checked in order):
- `.hookdeck/config.toml` (project-local)
- `~/.config/hookdeck/config.toml` (global)

## Quick Start

**1. Create a manifest file** (`hookdeck.jsonc`):

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "destination": {
    "name": "my-service",
    "url": "https://my-service.example.com/webhooks"
  }
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

## Manifest Guide

### Sources

Define a Hookdeck source to receive webhooks:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "source": {
    "name": "my-webhook-source",
    "description": "Receives order webhooks from external service"
  }
}
```

After deploying, the source URL from Hookdeck is automatically synced back to your `wrangler.jsonc` (disable with `--sync-wrangler=false`).

### Destinations

Define where events are delivered:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "destination": {
    "name": "order-processor",
    "url": "https://order-processor.example.com/webhooks",
    "rate_limit": 1,
    "rate_limit_period": "concurrent"
  }
}
```

### Connections

Wire a source to a destination with optional filtering:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "destination": {
    "name": "order-processor",
    "url": "https://order-processor.example.com/webhooks"
  },
  "env": {
    "staging": {
      "connection": {
        "name": "orders-to-processor",
        "source": "order-source",
        "filter": {
          "type": "order.created"
        }
      }
    }
  }
}
```

Connections reference sources and destinations by name. The `filter` shorthand is converted to a filter rule during deploy. Filters use a MongoDB-like query syntax with operators like `$and`, `$or`, and `$exist`:

```jsonc
"connection": {
  "name": "orders-to-processor",
  "source": "order-source",
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
  },
  "transformations": ["enrich-order"]
}
```

Both `filter` and `transformations` are shorthands that get converted to rules during deployment.

### Transformations

Transformations use the same schema as other manifests. Use `code_file` to point to a JavaScript file containing the transformation logic — its contents are read from disk and uploaded to Hookdeck on deploy:

```jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "transformation": {
    "name": "enrich-order",
    "description": "Adds computed fields to order payload",
    "code_file": "handler.js",
    "env": {
      "API_BASE_URL": "https://api.example.com"
    }
  },
  "env": {
    "production": {
      "transformation": {
        "env": {
          "API_BASE_URL": "https://api-production.example.com"
        }
      }
    }
  }
}
```

The `code_file` path is resolved relative to the manifest file. The JavaScript file must use the Hookdeck transformation signature:

```js
addHandler("transform", (request, context) => {
  // Access environment variables defined in the manifest
  const apiUrl = context.env.API_BASE_URL;

  // Modify the request before delivery
  request.body.processed_at = new Date().toISOString();
  return request;
});
```

Environment variables defined in `env` are available at runtime via `context.env`. Per-environment overrides (shown above) let you use different values across staging and production.

### Inheritance

Use `extends` to share configuration across manifests. A common pattern is a root manifest that defines environment profiles:

```jsonc
// Root hookdeck.jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "env": {
    "staging": { "profile": "staging" },
    "production": { "profile": "production" }
  }
}
```

```jsonc
// services/my-service/hookdeck.jsonc
{
  "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json",
  "extends": "../../hookdeck.jsonc",
  "destination": {
    "name": "my-service",
    "url": "https://my-service-dev.example.com"
  },
  "env": {
    "staging": {
      "destination": {
        "url": "https://my-service-staging.example.com"
      }
    },
    "production": {
      "destination": {
        "url": "https://my-service-production.example.com"
      }
    }
  }
}
```

Child manifests inherit all fields from the parent. Environment-specific overrides are merged on top.

### Environment Overrides

Deploy to specific environments with `--env`:

```bash
hookdeck-deploy deploy --env staging
hookdeck-deploy deploy --env production
```

The `env` object in the manifest defines per-environment overrides for any resource field:

```jsonc
{
  "destination": {
    "name": "my-service",
    "url": "https://my-service-dev.example.com"
  },
  "env": {
    "staging": {
      "destination": { "url": "https://my-service-staging.example.com" }
    },
    "production": {
      "destination": { "url": "https://my-service-production.example.com" }
    }
  }
}
```

Without `--env`, the base values are used (useful for local development).

### Variable Interpolation

Reference environment variables in manifest values with `${VAR_NAME}`:

```jsonc
{
  "destination": {
    "name": "my-service",
    "auth_type": "HOOKDECK_SIGNATURE",
    "auth": {
      "webhook_secret_key": "${HOOKDECK_SIGNING_SECRET}"
    }
  }
}
```

Variables are resolved from the process environment at deploy time.

## Project Structure

A recommended layout for monorepos with multiple webhook integrations:

```
hookdeck.jsonc              # Root manifest: environment profiles
sources/
  order-webhook/
    hookdeck.jsonc           # Source definition (extends root)
transformations/
  enrich-order/
    hookdeck.jsonc           # Transformation manifest (same schema, supports extends)
    handler.js               # Transformation code
destinations/
  order-processor/
    hookdeck.jsonc           # Destination + connection (extends root)
```

Each sub-manifest uses `extends` to inherit the root environment profiles, so you only define your profiles once. Transformation manifests use the same schema and can also use `extends`.

See the [`example/`](./example) directory for a working version of this layout.

### Deploy Scripts

A typical `package.json` setup for deploying across environments:

```json
{
  "scripts": {
    "deploy:staging": "hookdeck-deploy deploy -f sources/order-webhook/hookdeck.jsonc -e staging && hookdeck-deploy deploy -f transformations/enrich-order/hookdeck.jsonc -e staging && hookdeck-deploy deploy -f destinations/order-processor/hookdeck.jsonc -e staging",
    "deploy:production": "hookdeck-deploy deploy -f sources/order-webhook/hookdeck.jsonc -e production && hookdeck-deploy deploy -f transformations/enrich-order/hookdeck.jsonc -e production && hookdeck-deploy deploy -f destinations/order-processor/hookdeck.jsonc -e production",
    "drift:staging": "hookdeck-deploy drift -f destinations/order-processor/hookdeck.jsonc -e staging"
  }
}
```

Resources must be deployed in dependency order: sources and transformations before the destination/connection manifest that references them.

## CLI Reference

### Commands

| Command | Description |
|---------|-------------|
| `hookdeck-deploy deploy` | Upsert resources in dependency order (source -> transformation -> destination -> connection) |
| `hookdeck-deploy drift` | Compare manifest against live Hookdeck state, report missing or drifted resources |
| `hookdeck-deploy status` | Show whether each manifest resource exists on Hookdeck with name, ID, and URL |
| `hookdeck-deploy schema [type]` | Output JSON schema for manifest files (`deploy` or `transformation`) |

### Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--file <path>` | `-f` | Manifest file path (default: `hookdeck.jsonc` or `hookdeck.json`) |
| `--env <name>` | `-e` | Environment overlay (e.g., `staging`, `production`) |
| `--dry-run` | | Preview changes without applying |
| `--profile <name>` | | Override credential profile |

### Deploy Flags

| Flag | Description |
|------|-------------|
| `--sync-wrangler` | Sync source URL back to `wrangler.jsonc` after deploy (default: `true`) |

## JSON Schemas

Add a `$schema` property to your manifest for IDE autocompletion and validation:

```jsonc
// For all manifests (sources, destinations, connections, transformations):
{ "$schema": "node_modules/@toppy/hookdeck-deploy-cli/schemas/hookdeck-deploy.schema.json" }
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
