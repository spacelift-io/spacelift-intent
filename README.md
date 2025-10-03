# github.com/spacelift-io/spacelift-intent MCP

Welcome to the Spacelift Intent open-source project! Intent is an MCP Server, that lets infrastructure engineers describe what they need in natural language and provisions it directly through calling provider APIs — skipping the Terraform/OpenTofu configuration layer entirely. It’s early and experimental, so expect rough edges, but that’s where you come in: try it out, tell us what works (and what doesn’t), and join the conversation in GitHub Discussions. This repo hosts the open-source core; there's also a fully managed version, built into Spacelift Platform - check out Spacelift Intent HERE.

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
