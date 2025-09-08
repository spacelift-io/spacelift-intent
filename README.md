# Squirrel-mcp

## Development

### Stdio mode - standalone

1. Generate and build project

```bash
go generate ./... && make build-standalone
```

2. Add server to you MCP host, e.g. to Claude Code:

```bash
claude mcp add squirrel-mcp -- `pwd`/bin/squirrel-standalone
```

### HTTP mode - with Spacelift as middleware

1. Create your own `.env` file following `.env.template`
2. Start containers

```bash
docker compose up
```

3. Add server to your MCP host, e.g. to Claude Code:

```bash
claude mcp add --transport http squirrel-mcp http://localhost:11995/mcp
```

or Claude Desktop (Settings -> Developer -> Edit Config)


```json
    "intent-infra-mcp": {
      "command": "npx",
      "args": [
        "--yes",
        "mcp-remote@0.1.27",
        "http://localhost:11995/mcp"
      ]
    }
```

#### Oauth

To enable OAuth authorization, you need to remove remove `SPACELIFT_API_KEY_ID`
and `SPACELIFT_API_KEY_SECRET` from your `.env` file and rebuild the image.
Remember that OAuth works with `http` server only.

### HTTP mode - standalone

1. Create your own `.env` file following `.env.template`. You can ignore `SPACELIFT_URL` property
2. (Optional) If you wish to use AWS provider, create your own `.env.aws` file following the `.env.aws.template`.
3. Start containers

```bash
docker compose -f docker-compose.standalone.yml up
```

4. Add server to your MCP host, e.g. to Claude Code:

```bash
claude mcp add --transport http squirrel-mcp http://localhost:11995/mcp
```

### Enabling Datadog traces collection

1. Create `docker-compose.override.yml` file in `local.dev` repository to expose DataDog agent port to localhost.
```yaml
services:
  dd-agent:
    ports:
      - "8126:8126"

```
2. Uncomment / Set observability related env variables, in your `.env` file
```
OBSERVABILITY_VENDOR=datadog
DD_AGENT_HOST=localhost
```
3. Run docker compose as described in [HTTP mode - with Spacelift as middleware](#http-mode---with-spacelift-as-middleware)

## TODOs

### Functionality

- [ ] Handling of provider configuration (other than just envvar);
- [ ] Namespaces;
- [ ] Multiple providers;
- [ ] Collaboration using Postgres server instead of SQLite. Where can that take us? Postgres is almost an app in itself;
- [ ] User management. Idea: Postgres user permissions;
- [ ] Encryption;
- [ ] Handling of sensitive values;

### Refactor

- [ ] opContext can be part of context.Context IMO
- [ ] History should also cover the policies;
