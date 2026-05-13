# Provider Allowlist Support

**Issue**: [#56](https://github.com/spacelift-io/spacelift-intent/issues/56)
**Status**: Design — awaiting maintainer sign-off before implementation

## Problem

The MCP server currently accepts any provider name the LLM hands it. Trust decisions sit entirely in chat between the user and the model — a typosquatting / lookalike provider (e.g. `kuwas/github`) can slip in alongside `hashicorp/aws` and `integrations/github`. Prompt-level guidance is unreliable.

We want a deployer-controlled mechanism that constrains which providers the server is willing to download, configure, and operate against.

## Non-goals

- Mutating the allowlist from inside the agent (the LLM is the entity being constrained — it must not be able to widen its own trust boundary).
- Supply-chain *integrity* (shasum pinning per version) — separate feature.
- Flipping the default to deny-unless-allowlisted — would be a breaking change; not planned. Backward compatibility is preserved indefinitely: unset flag → unrestricted.

## Design summary

| Decision | Choice |
|---|---|
| Surface | `--provider-allowlist-file <path>` (env: `PROVIDER_ALLOWLIST_FILE`) |
| Default (flag unset) | All providers allowed; one-line startup warning. Backward compatible. |
| Builtin set | `--provider-allowlist-file=builtin:trusted` → embedded YAML |
| Builtin contents | `hashicorp/*` + `opentofu/*` only |
| File format | YAML |
| Name matching | Exact (`namespace/type`) or namespace wildcard (`namespace/*`) |
| Version matching | Optional `versions:` per entry, Terraform `required_providers` constraint syntax via `hashicorp/go-version` |
| Multi-entry semantics | Most-specific-wins (exact > namespace wildcard); OR-of-rules within the same specificity tier |
| Enforcement | (1) hard gate in `LoadProvider`; (2) `RegistryClient` decorator filtering search results |
| Missing / malformed file | Fail-closed at startup |
| MCP tools to mutate allowlist | None |

## Why a file rather than MCP tools

The allowlist is a trust boundary, and the LLM is the entity we're trying to constrain. Giving the agent tools to mutate its own allowlist defeats the point. The deployer (human) owns the file; the agent can only consult it.

The file path is set by the deployer. No default path is auto-discovered — explicit only, to avoid "why did my container pick up my laptop's allowlist?" surprises.

## File format

```yaml
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0, < 7.0.0"   # optional; same syntax as required_providers
  - name: hashicorp/random          # no constraint → any version
  - name: hashicorp/*               # namespace wildcard
  - name: integrations/github
    versions: "~> 6.0"
```

### Name patterns

- **Exact**: `hashicorp/aws` — matches that provider only.
- **Namespace wildcard**: `hashicorp/*` — matches any name in that namespace.
- **Not supported in v1**: cross-namespace wildcards (`*/aws`), arbitrary globs, regex.

### Version constraints

- Use `github.com/hashicorp/go-version` — same library and syntax Terraform/OpenTofu use for `required_providers`. Users don't have to learn a second mini-language.
- Supported forms: `>= 1.2.0`, `~> 1.2`, `>= 1.0, < 2.0`, etc.
- Omitting `versions:` means "any version".
- Constraints do **not** inherit between rules — each entry is self-contained.

### Match semantics: most-specific-wins

Specificity tiers (highest to lowest):

1. Exact name match.
2. Namespace wildcard.

For a given provider request:

1. Collect all entries whose name pattern matches.
2. Pick the highest specificity tier with at least one matching entry.
3. Within that tier, OR the version constraints — the provider is allowed if **any** entry in the chosen tier permits the version.

This matches the model people already know from firewalls, routing tables, CSS, `.gitignore`.

#### Examples

```yaml
providers:
  - name: hashicorp/*
  - name: hashicorp/aws
    versions: ">= 5.0.0"
```

| Request | Matched entries | Chosen tier | Decision |
|---|---|---|---|
| `hashicorp/aws@4.50` | wildcard, exact | exact | **denied** (4.50 fails `>= 5.0.0`) |
| `hashicorp/aws@5.20` | wildcard, exact | exact | **allowed** |
| `hashicorp/random@3.0` | wildcard only | wildcard | **allowed** (no constraint) |
| `kuwas/github@4.3.0` | none | — | **denied** |

```yaml
providers:
  - name: hashicorp/aws
    versions: ">= 5.0.0"
  - name: hashicorp/aws
    versions: "< 4.0.0"
```

Both entries are at "exact" tier → OR within tier.

| Request | Decision |
|---|---|
| `hashicorp/aws@3.5` | **allowed** (matches second rule) |
| `hashicorp/aws@4.5` | **denied** (matches neither) |
| `hashicorp/aws@5.2` | **allowed** (matches first rule) |

```yaml
providers:
  - name: hashicorp/*
    versions: ">= 2.0.0"
  - name: hashicorp/aws
```

The exact entry has no version constraint. **The wildcard's constraint does not leak down.**

| Request | Decision |
|---|---|
| `hashicorp/aws@1.5` | **allowed** (exact tier wins; no constraint = any) |
| `hashicorp/random@1.5` | **denied** (only wildcard matches; 1.5 fails `>= 2.0.0`) |
| `hashicorp/random@2.5` | **allowed** |

## Builtin trusted set

Shipped as an embedded YAML file (`//go:embed`). Selected via `--provider-allowlist-file=builtin:trusted`.

```yaml
# allowlist/builtin/trusted.yaml
providers:
  - name: hashicorp/*
  - name: opentofu/*
```

Rationale: both namespaces are first-party / well-known maintainers. `opentofu/*` is largely a re-host of `hashicorp/*` (post-license-change), so trusting both covers both tofu and terraform users. `integrations/*` (one provider: `github`) and `spacelift-io/*` are deliberately left out — users opt in via their own file.

The `builtin:` prefix is a sentinel parsed by the loader; the rest of the load path is identical to a file path.

## Enforcement layers

Two layers — the first is the real gate, the second is UX.

### 1. Hard gate in `OpenTofuAdapter.LoadProvider`

Every resource / data-source lifecycle tool funnels through `LoadProvider` (`provider/opentofu_adapter.go:51`) before it can do anything with a provider. Inject the allowlist there and check **before** `loadProvider` runs — we don't want to download a binary just to reject it. Concrete version is known at this point, so version-constraint enforcement happens here.

On denial: return an error like
> `provider hashicorp/foo@1.2.3 not in allowlist`

…surfaced verbatim through `i.NewToolResultError` so the model can tell the user *why*, rather than retrying around a generic failure.

### 2. `RegistryClient` decorator (UX layer)

Wrap `RegistryClient` to filter results from `SearchProviders` / `FindProvider` so disallowed providers don't surface to the model. Search results don't carry a concrete version, so this layer only enforces *name* membership; version constraints are enforced at load time.

```go
// registry/allowlist_client.go
func NewAllowlistedClient(inner types.RegistryClient, al allowlist.Allowlist) types.RegistryClient
```

Wired in `cmd/spacelift-intent/server.go:newServer`.

### Server instructions

When an allowlist is configured, append a short summary to `instructions.Get()` so the model is told upfront which providers are usable. Reduces wasted tool calls on disallowed providers.

## Failure modes

| Situation | Behavior |
|---|---|
| Flag unset | Allowlist disabled. One-line startup warning: *"provider allowlist disabled — all providers permitted"*. |
| Flag set, file missing | Fail-closed at startup with a clear error. |
| Flag set, malformed YAML | Fail-closed at startup. |
| Flag set, invalid version constraint | Fail-closed at startup, identify offending entry. |
| Flag set, no entries | Treat as "deny all", with a startup warning to make this obvious. |

Fail-closed is deliberate: a typo in the flag path silently disabling the gate would be a footgun.

## Dependencies added

- `github.com/hashicorp/go-version` — Terraform's own constraint library; matches `required_providers` syntax.
- `gopkg.in/yaml.v3` — already on the transitive dependency tree, promoted to direct.

## File-touch list

| File | Change |
|---|---|
| `allowlist/allowlist.go` (new) | Core type, YAML loader, matcher (name + version + specificity tiers), builtin embed |
| `allowlist/allowlist_test.go` (new) | Matching unit tests covering tier precedence, OR-within-tier, all edge cases above |
| `allowlist/builtin/trusted.yaml` (new) | Embedded default set |
| `registry/allowlist_client.go` (new) | Decorator filtering `SearchProviders` / `FindProvider` |
| `registry/allowlist_client_test.go` (new) | Decorator filter tests |
| `provider/opentofu_adapter.go` | Inject `Allowlist`; gate at top of `LoadProvider` |
| `provider/manager_test.go` | Test asserting `LoadProvider` short-circuits before download on denial |
| `cmd/spacelift-intent/flags.go` | New `providerAllowlistFileFlag` |
| `cmd/spacelift-intent/main.go` | Load allowlist, pass to `Config` |
| `cmd/spacelift-intent/server.go` | Wire decorator + adapter |
| `instructions/instructions.go` | Append allowlist summary when configured |
| `README.md` | Document flag, file syntax, builtin set |
| `go.mod` | Add `hashicorp/go-version`; promote `yaml.v3` |

Estimated ~500 LOC including tests.

## Test plan

Unit tests in `allowlist/`:

- Exact name match.
- Namespace wildcard match.
- No match → deny.
- Version constraint pass / fail / absent.
- Tier precedence: exact overrides wildcard.
- OR within tier (multiple exact entries for same name).
- Constraint non-inheritance (wildcard constraint must not leak to exact rule).
- Disabled allowlist passes everything.
- Malformed YAML fails to load.
- Invalid version constraint fails to load.

Integration:

- `provider/manager_test.go`: mock registry, assert `LoadProvider` short-circuits before download when name or version is denied; assert error message contains the denial reason.
- `registry/allowlist_client_test.go`: search results are filtered; `FindProvider` returns nil for denied names.

## Open questions for maintainer sign-off

1. **Builtin set content** — confirmed: `hashicorp/*` + `opentofu/*`. Long-abandoned `hashicorp/*` providers (e.g. `arukas`, `dyn`, `oneandone`) are included by namespace wildcard; not a real supply-chain risk since they receive no updates of any kind, but worth noting.
2. **Match semantics** — most-specific-wins, OR within tier. Explained with examples above.

## Out of scope (future work, not v1)

- Cross-namespace wildcards (`*/aws`).
- Shasum / checksum pinning.
- Allowlist hot-reload (currently load-at-startup only).
- A "block list" mode (allowlist is clearer and safer by default).
- Per-resource-type granularity (e.g. trust `hashicorp/aws` resources but not data sources).

## Implementation order

1. `allowlist/` package + tests (no wiring yet).
2. `allowlist/builtin/trusted.yaml` + embed.
3. `RegistryClient` decorator + tests.
4. `LoadProvider` gate + tests.
5. CLI flag + `cmd/spacelift-intent/` wiring.
6. Instructions append.
7. README documentation.

Each step compiles and tests on its own. Final PR sequences these as commits.
