# charter

<p align="center"><img src="charter-logo.png" alt="charter logo" width="200"></p>

> Turn fuzzy intent into hardened, machine-readable specifications.

CHARTER is an upstream companion to [`acig`](https://github.com/helloodokai/acig) that turns half-written GitHub issues, Slack threads, and "hey can you…" messages into versioned, machine-readable contracts that coding agents consume as specifications. Where `acig` verifies output **after** an agent finishes work, CHARTER hardens intent **before** the agent starts.

## Install

```bash
# macOS / Linux (arm64)
curl -sSL https://github.com/helloodokai/charter/releases/latest/download/charter_darwin_arm64.tar.gz \
  | tar -xz -C /usr/local/bin charter

# Linux (amd64)
curl -sSL https://github.com/helloodokai/charter/releases/latest/download/charter_linux_amd64.tar.gz \
  | tar -xz -C /usr/local/bin charter

# Homebrew
brew tap helloodokai/charter-tap
brew install charter

# Or build from source:
git clone https://github.com/helloodokai/charter.git
cd charter && make build && cp dist/charter /usr/local/bin/
```

## Quick start

Draft your first charter from a GitHub issue in under a minute:

```bash
# Set your Ollama Cloud API key
export OLLAMA_API_KEY=your-key-here

# Draft a charter from any GitHub issue
charter draft --issue https://github.com/your-org/your-repo/issues/42

# Or from a file
charter draft --from requirements.md

# Or pipe from stdin
echo "Add a rate limiter to the API" | charter draft --stdin
```

CHARTER runs an **interactive Socratic dialogue** — it asks you one question at a time, proposes answers based on context, and progressively hardens the spec until it's tight enough to hand to a coding agent.

When the dialogue finishes, you get a `charter.yaml` file in `.charters/`:

```yaml
schema_version: "1"
id: ch-2026-05-04-a1b2c3
created_at: 2026-05-04T12:00:00Z
updated_at: 2026-05-04T12:08:00Z
authors:
  - you
source:
  type: github_issue
  url: https://github.com/org/repo/issues/42
goal: Add a rate limiter to the public API
context: The API currently has no rate limiting and is vulnerable to abuse
non_goals:
  - This charter does NOT change the authentication layer
  - This charter does NOT add caching
acceptance_criteria:
  - id: ac-1
    statement: API returns 429 when rate limit is exceeded
    verification: test
  - id: ac-2
    statement: Rate limits are configurable per route
    verification: test
edge_cases:
  - What happens when a burst exceeds the limit by 10x?
blast_radius:
  files:
    - src/api/**"
constraints:
  performance:
    - p99 latency must stay under 100ms
risk: medium
risk_rationale: Touches public API surface but scoped to one concern
status: ready
```

## Setting up OLLAMA_API_KEY

CHARTER uses [Ollama Cloud](https://ollama.com) as its primary LLM backend. Get an API key at [ollama.com](https://ollama.com):

```bash
export OLLAMA_API_KEY=ollama-your-key-here
```

Optional — for the frontier counter-spec pass (recommended):

```bash
export ANTHROPIC_API_KEY=sk-ant-your-key-here
```

Verify your setup:

```bash
charter doctor
```

## CLI reference

```
charter draft [--issue URL] [--from FILE] [--stdin] [--out PATH] [--non-interactive] [--turn-budget N] [--profile cloud|local] [--resume ID]
charter validate <charter.yaml>
charter conformance <charter.yaml> --diff REF..REF [--format json|md|both] [--out PATH]
charter ls [--status draft|ready|approved|archived]
charter approve <charter.yaml>
charter schema
charter doctor
charter app serve [--addr :8080]
```

### `charter draft`

The workhorse. Starts an interactive Socratic dialogue to produce a charter.

| Flag | Description |
|------|-------------|
| `--issue URL` | Source from a GitHub issue |
| `--from FILE` | Source from a local file |
| `--stdin` | Source from stdin |
| `--out PATH` | Output path (defaults to `.charters/<id>.yaml`) |
| `--non-interactive` | Run in CI mode without interactive prompts |
| `--turn-budget N` | Override default turn budget (default: 8) |
| `--profile cloud\|local` | Model profile to use |
| `--resume ID` | Resume an interrupted dialogue |
| `--no-transcript` | Omit transcript from output |

### `charter validate`

Checks a charter for completeness and internal consistency. Reports field-level errors with helpful messages. Exit code 0 = valid, 1 = issues found.

### `charter conformance`

Grades a git diff against a charter. Produces structured findings in the same format `acig` consumes, so integration is seamless:

```bash
charter conformance .charters/ch-2026-05-04-abc123.yaml --diff main..feature
```

Graders:
- **Goal alignment** — does the diff implement the goal?
- **Acceptance criteria coverage** — are all criteria met?
- **Non-goal violations** — does the diff touch excluded areas?
- **Blast radius compliance** — does the diff stay within declared paths?
- **Unknown gating** — any blocking unknowns block conformance

### `charter approve`

Transitions a charter from `ready` to `approved`. Approved charters are immutable — they represent a contract.

### `charter ls`

List charters, optionally filtered by status.

### `charter schema`

Emit the JSON schema for `charter.yaml`. Useful for IDE integration and validation tooling.

### `charter doctor`

Check backend reachability and configuration. Verifies API keys and connectivity to all configured LLM backends.

## GitHub App setup

CHARTER can run as a GitHub App that automates the dialogue inside issue threads.

### Secrets

```yaml
# .github/workflows/charter.yml or your app environment
secrets:
  OLLAMA_API_KEY: ${{ secrets.OLLAMA_API_KEY }}
  ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}  # optional
  GITHUB_APP_ID: ${{ secrets.CHARTER_APP_ID }}
  GITHUB_APP_PRIVATE_KEY: ${{ secrets.CHARTER_APP_PRIVATE_KEY }}
```

### App mode

```bash
charter app serve --addr :8080
```

The app listens for `issues` webhooks. When an issue is labeled `needs-charter`, it starts a dialogue in the issue thread. When the dialogue completes, it opens a PR with the charter file and tags the issue with `has-charter`.

### Action mode

The reusable workflow at `.github/workflows/charter.yml` runs `charter draft --issue <url> --non-interactive` when an issue is labeled `needs-charter`. It creates a PR with the draft charter for human review.

## How charter integrates with acig

CHARTER and `acig` are siblings in the same pipeline:

```
  Issue ──► CHARTER ──► charter.yaml ──► Agent ──► PR ──► acig
           (harden)     (contract)      (code)    (diff)  (verify)
```

In `acig`, reference a charter in your PR body or commit message:

```
Charter: .charters/ch-2026-05-04-abc123.yaml
```

`acig` loads the charter, runs `charter conformance`, and merges the findings into its verdict.

## The charter.yaml schema

The schema is versioned (`schema_version: "1"`). Key fields:

| Field | Description |
|-------|-------------|
| `id` | Human-friendly ID: `ch-YYYY-MM-<hex>` |
| `goal` | One-sentence statement of intent |
| `context` | Background an agent needs |
| `non_goals` | Explicit scope exclusions |
| `acceptance_criteria` | Testable conditions for success |
| `edge_cases` | Scenarios the agent must handle |
| `constraints` | Performance, security, compatibility, style, dependency limits |
| `blast_radius` | Files, services, data stores the change may touch |
| `unknowns` | Open questions (blocking or not) |
| `counter_spec` | Ways an agent could misinterpret the goal |
| `risk` | low / medium / high / critical |
| `status` | draft / ready / approved / archived |

## For AI agents consuming charters

If you're building an agent that reads `charter.yaml`, here's what to focus on:

1. **`goal`** is the single source of truth. Everything else elaborates on it.
2. **`non_goals`** are hard constraints. If your diff touches areas listed here, conformance will fail.
3. **`acceptance_criteria`** are your test plan. Each criterion has a `verification` method (test, manual, or metric).
4. **`blast_radius`** defines your allowed scope. Stay within it.
5. **`unknowns`** with `blocking: true` must be resolved before work starts.
6. **`counter_spec`** documents known misinterpretation risks — read these before starting.
7. **`constraints`** are non-negotiable. Violating them fails conformance.

The JSON schema is available at `schema/charter.schema.json` or via `charter schema`.

## Configuration

Create `.charter.toml` in your repo root (optional, sensible defaults exist):

```toml
[dialogue]
turn_budget        = 8
ask_for_rollback_at = "high"
require_counter_spec = true

[storage]
charters_dir = ".charters"

[models]
default_profile   = "cloud"
fallback_to_local = true

[models.profiles.cloud]
cheap    = { provider = "ollama_cloud", name = "gpt-oss:20b" }
mid      = { provider = "ollama_cloud", name = "qwen3-coder:480b" }
frontier = { provider = "anthropic",    name = "claude-sonnet-4-6" }

[models.profiles.local]
cheap    = { provider = "ollama_local", name = "qwen2.5-coder:7b" }
mid      = { provider = "ollama_local", name = "qwen2.5-coder:32b" }
frontier = { provider = "anthropic",    name = "claude-sonnet-4-6" }

[models.ollama_cloud]
host = "https://ollama.com"
api_key = "${OLLAMA_API_KEY}"

[github]
app_id_env      = "GITHUB_APP_ID"
private_key_env = "GITHUB_APP_PRIVATE_KEY"
needs_label     = "needs-charter"
has_label       = "has-charter"

[paths]
critical = ["src/auth/**", "src/payments/**", "migrations/**"]
```

Charters touching `paths.critical` are auto-bumped to at least `risk = high`, which forces a rollback plan.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Validation/conformance issues found, non-blocking |
| 2 | Charter rejected (incomplete, or diff materially violates charter) |
| ≥10 | Tool error |

## Building from source

```bash
git clone https://github.com/helloodokai/charter.git
cd charter
make build
```

Run tests:

```bash
make test
make vet
make lint  # requires golangci-lint
```

## License

MIT