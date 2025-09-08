# github.com/spacelift-io/spacelift-intent MCP

## Development

### Stdio mode

1. Generate and build project

```bash
go generate ./... && make build-standalone
```

2. Add server to you MCP host, e.g. to Claude Code:

```bash
claude mcp add github.com/spacelift-io/spacelift-intent -- `pwd`/bin/github.com/spacelift-io/spacelift-intent
```
