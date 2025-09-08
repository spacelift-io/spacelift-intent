# Repository Guidelines

## Project Structure & Module Organization
- `cmd/spacelift-intent/` hosts the CLI entrypoint that wires flags, creates the server, and handles shutdown.
- `tools/` groups MCP tool handlers; see `tools/provider`, `tools/lifecycle`, and `tools/state` for schema, lifecycle, and state actions.
- Core integrations live in `provider/`, `registry/`, `storage/`, and `types/`, while Terraform plugin definitions sit in `proto/` with generated code under `generated/tfplugin5/`.
- Integration scenarios live in `test/` (CloudFront, RabbitMQ, SQLite helpers); package-scoped tests sit alongside implementation files.

## Build, Test, and Development Commands
- `go generate ./...` refreshes protobuf bindings prior to builds.
- `make build` emits `bin/spacelift-intent`; use `make build-legacy` when targeting the legacy plugin tag.
- `make test` executes unit tests; `go test ./test -v` runs integration suites, often with `GOLANG_PROTOBUF_REGISTRATION_CONFLICT=warn`.
- `make lint`, `make fmt`, and `make deps` wrap linting, formatting, and dependency syncing.
- `make run` starts the server with existing binaries; `make run-with-env` sources `.env.aws` before launch.

## Coding Style & Naming Conventions
- Follow standard Go formatting enforced by `go fmt` and `golangci-lint`; commit only gofmt’d files.
- Keep packages lower-case nouns; exported identifiers PascalCase; wrap errors with `%w`.
- Name tests `TestXxx`/`BenchmarkXxx`; favor table-driven cases where practical.
- Align configuration structs with Terraform schema names to keep provider terminology consistent.

## Testing Guidelines
- Default to `make test`; prefer `go test ./... -shuffle=on` when adding logic-heavy flows.
- Integration runs in `test/` may need `USE_OPENTOFU_PROVIDER_LIB=true` and extended timeouts (for example `-timeout 1000s` for CloudFront).
- Keep fixtures in `test/testhelper`; ensure helpers tear down external resources.
- Aim for coverage on new paths and document any gaps in the PR body.

## Commit & Pull Request Guidelines
- Use short, imperative commit subjects similar to `Add spacelift provider tests`, optionally appending the PR number in parentheses.
- Rebase or squash before raising a PR to keep history linear.
- PRs should state intent, link issues, list test commands, and add evidence for UX or API changes.
- Flag configuration changes (new flags, env vars) in the PR description and alert reviewers when credentials are required.

## Security & Configuration Tips
- Store AWS credentials in `.env.aws`; never commit secrets and prefer environment-specific profiles.
- Provide durable paths for `--tmp-dir` and `--db-dir` when running standalone to preserve cached state.
- Exercise new provider features against `storage/sqlite.go` and note extra persistence needs.
