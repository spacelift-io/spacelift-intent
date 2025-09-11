# github.com/spacelift-io/spacelift-intent MCP

## Development

### Stdio mode

1. Generate and build project

```bash
go generate ./... && make build
```

2. Add server to your MCP host

### Claude Desktop Configuration

Add the following configuration to your Claude Desktop config file:

**macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
**Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
**Linux**: `~/.config/claude/claude_desktop_config.json`

```json
{
    "mcpServers": {
        "spacelift-intent": {
            "command": "/path/to/your/spacelift-intent/bin/spacelift-intent",
            "args": [
                "--tmp-dir",
                "/path/to/your/spacelift-intent/tmp",
                "--db-dir",
                "/path/to/your/spacelift-intent/db"
            ]
        }
    }
}
```

**Note**: Replace `/path/to/your/spacelift-intent/` with the actual path to your project directory.

### Claude Code (Alternative)

```bash
claude mcp add github.com/spacelift-io/spacelift-intent -- `pwd`/bin/spacelift-intent
```
