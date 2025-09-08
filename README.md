# Spacelift-Intent MCP

## Development

### Stdio mode

1. Generate and build project

```bash
go generate ./... && make build-standalone
```

2. Add server to you MCP host, e.g. to Claude Code:

```bash
claude mcp add spacelift-intent -- `pwd`/bin/spacelift-intent
```
