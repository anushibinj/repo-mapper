# Repo Mapper

Repo Mapper is an **AI-first repository architecture mapper**. It scans a
codebase, deterministically parses the languages/frameworks it recognizes
(Java, Spring, React, Vite, Node, SQL, Docker), and generates a living set of
Markdown, Mermaid, and JSON documentation describing the repository's
modules, components, routes, APIs, database tables, and features.

The generated output (`.repo-mapper/` by default) is meant to be committed
to the repository and used as the **entrypoint for both humans and LLMs** to
understand the codebase quickly — instead of an LLM having to deep-dive
through the entire source tree every session, it can start from
`.repo-mapper/README.md` and the accompanying JSON maps.

See [`PRD.md`](PRD.md) for the full product/design spec this project was
built from, including rationale for every major decision.

## How it works, briefly

1. **Scan** — walk the repository, respecting `.gitignore` and configured
   excludes, hashing each file.
2. **Parse** — dispatch each file to whichever registered plugin(s) claim
   it (by extension/filename), producing structural `Entity` records. A
   local `.cache/` (never committed) stores per-file hashes and parsed
   entities so unchanged files are never re-parsed.
3. **Analyze** — turn raw entities into the canonical `Repository` model:
   components, routes, APIs, tables, and inferred features (groupings that
   span frontend/backend/database for the same business capability).
4. **Generate** — render that model to Markdown, Mermaid diagrams, and JSON
   into the output directory.
5. **(Optional) LLM summarization** — if configured, an OpenAI-compatible
   endpoint is used strictly for prose (summaries/descriptions), never for
   extracting structural facts that deterministic parsing already provides.

## Installation / build

Requires **Go 1.26+** and `git` on `PATH`.

```powershell
git clone https://github.com/anushibinj/repo-mapper.git
cd repo-mapper
go build -o bin\repo-mapper.exe .\cmd\repo-mapper
```

On macOS/Linux:

```bash
go build -o bin/repo-mapper ./cmd/repo-mapper
```

Verify the build and environment:

```powershell
.\bin\repo-mapper.exe doctor
```

## Usage

All commands accept `-root <path>` to point at the repository to analyze
(defaults to the current directory).

```
repo-mapper <command> [flags]
```

| Command   | Purpose                                                                 |
|-----------|--------------------------------------------------------------------------|
| `scan`    | Full repository scan; writes complete documentation to the output dir.  |
| `update`  | Incremental — uses `git diff` to re-parse only changed files.          |
| `clean`   | Deletes the local `.cache/` directory.                                  |
| `doctor`  | Validates git availability, repo root, config, output dir writability. |
| `plugins` | Lists registered plugins and whether each is enabled.                  |
| `graph`   | Prints a single Mermaid diagram to stdout without writing any files.    |
| `export`  | Prints (or writes) the canonical JSON model — useful for scripting.    |

Run `repo-mapper <command> -h` for command-specific flags.

### `scan` — full scan

Walks the entire repository and regenerates all output. Use this the first
time, or whenever you want a from-scratch rebuild.

```powershell
.\bin\repo-mapper.exe scan -root .
```

### `update` — incremental update

Uses `git diff` to find changed files and only re-parses those, reusing
cached entities for everything else. Much faster than `scan` on large repos
and is the command intended for CI/CD (see below).

```powershell
.\bin\repo-mapper.exe update -root .
.\bin\repo-mapper.exe update -root . -base origin/master
```

**Base ref resolution**, when `-base` is not given:
1. If there are uncommitted working-tree changes, diff those against HEAD
   (the normal local dev-loop experience).
2. Otherwise (a **clean checkout — the typical CI state**), auto-detect the
   base by reading the commit hash recorded in the last generated
   `repo-map.json` and diff from there to `HEAD`. This is what makes
   `repo-mapper update` (with zero flags) work correctly right after a
   fresh CI checkout — a plain `git diff` would otherwise see no
   differences and re-process nothing.
3. If that recorded commit is no longer reachable (shallow clone, rebased
   history, squashed merge), `update` logs a warning and transparently
   falls back to a full `scan` instead of failing or silently skipping
   everything. An explicitly passed `-base` is always honored as-is; if
   *that* fails to resolve, the error is surfaced normally.

**Exit codes** (important for automation):
| Code | Meaning                                                              |
|------|-----------------------------------------------------------------------|
| `0`  | Ran successfully and wrote real (architectural) changes.             |
| `3`  | Ran successfully but nothing changed except git commit metadata — output was **not** rewritten. |
| `1`  | A real error occurred.                                                |

`repo-map.json` embeds the current commit hash/branch, so a naive "always
commit the output" CI step would create an infinite commit loop (the SHA
always differs after the previous commit). `scan`/`update` detect this case
and skip the rewrite — see `PRD.md` section 20 for the full explanation and
a sample CI snippet.

### `graph` — print one diagram

```powershell
.\bin\repo-mapper.exe graph -root . -diagram system
# -diagram: system | backend | frontend | database | auth
```

### `export` — canonical JSON model

```powershell
.\bin\repo-mapper.exe export -root . -out repo.json
# omit -out to print to stdout, e.g. for piping into jq
```

### `clean` — clear the cache

```powershell
.\bin\repo-mapper.exe clean -root .
```

### `doctor` — environment check

```powershell
.\bin\repo-mapper.exe doctor -root .
```

### `plugins` — list plugins

```powershell
.\bin\repo-mapper.exe plugins -root .
```

## Configuration

Repo Mapper works with zero configuration. To customize behavior, add a
`repo-mapper.yaml` file to the repository root:

```yaml
scan:
  exclude:
    - node_modules
    - build
    - dist
    - target
    - .git
    - .repo-mapper
    - .cache
  ignoreFile: ""   # optional extra ignore file, .gitignore-style
  workers: 0       # 0 = runtime.NumCPU() * 2

output:
  directory: .repo-mapper

llm:
  enabled: false
  endpoint: ""
  api_key: ""
  model: ""

plugins:
  java: true
  spring: true
  react: true
  vite: true
  node: true
  docker: true
  sql: true
```

Any field you omit falls back to the default shown above.

## What gets committed vs. ignored

- **`.repo-mapper/`** (or whatever `output.directory` is set to) — commit
  this. It's the generated documentation and is the intended entrypoint for
  both humans and a fresh LLM session (start with
  `.repo-mapper/README.md`).
- **`.cache/`** — never commit this. It's a pure performance cache
  (per-file hashes + parsed entities) that churns on every run and has no
  reader value. Repo Mapper automatically ensures `.cache/` is covered by
  your `.gitignore` on every `scan`/`update` — no manual setup required.

## CI/CD

A typical pipeline: checkout → `repo-mapper update` → commit the output
(only if it actually changed) → push.

```bash
./repo-mapper update
case $? in
  0) git add .repo-mapper && git commit -m "docs: update repo map" && git push ;;
  3) echo "no changes to commit" ;;
  *) exit 1 ;;
esac
```

## GitHub Actions CI integration

Repo Mapper ships a reusable GitHub Actions workflow that you can call
directly from any repository — no copy-pasting required.

### Minimal setup

Create `.github/workflows/repo-map.yml` in **your repository** with the
following content. All inputs are optional — this is all you need:

```yaml
name: Repo Map

on:
  push:
    branches: [master]      # adjust to your default branch

jobs:
  repo-map:
    permissions:
      contents: write     # required: the job pushes the updated docs back
    uses: anushibinj/repo-mapper/.github/workflows/update-repo-map.yml@master
```

### Full example with all inputs

```yaml
name: Repo Map

on:
  push:
    branches: [master]      # (mandatory in your workflow) branch(es) to watch
  workflow_dispatch:      # optional: allow manual runs from the Actions tab

jobs:
  repo-map:
    permissions:
      contents: write     # (mandatory) allows the job to push the updated docs

    uses: anushibinj/repo-mapper/.github/workflows/update-repo-map.yml@master
    with:
      # ── optional inputs (shown with their defaults) ──────────────────────

      # Version of repo-mapper to install.
      # Use 'latest' to always get the newest release, or pin to a tag
      # (e.g. 'v0.2.0') for reproducible builds.
      repo-mapper-version: latest

      # Go toolchain version used only to install repo-mapper.
      # 'stable' always resolves to the current stable Go release.
      go-version: stable

      # Commit message written when the docs are updated.
      # '[skip ci]' prevents this commit from triggering another workflow run.
      commit-message: 'chore: update repo-mapper documentation [skip ci]'

      # Git author identity for the automated commit.
      committer-name: github-actions[bot]
      committer-email: github-actions[bot]@users.noreply.github.com
```

### How the workflow behaves

1. **Checks out** your repository with full history (`fetch-depth: 0`) so
   the incremental update can diff against the commit hash recorded in the
   last generated `.repo-mapper/repo-map.json`.
2. **Installs** `repo-mapper` from the Go module proxy — no binary download
   or pre-built asset required.
3. **Runs** `repo-mapper update`. Exit code `3` ("no architectural changes")
   is treated as success; only exit code `1` fails the job.
4. **Commits and pushes** any changed files under `.repo-mapper/`,
   `.github/copilot-instructions.md`, and `.github/skills/`. If nothing
   changed, the commit step is skipped entirely (no empty commits).

> **Note on push protection** — if your default branch has required status
> checks or branch protection rules that block direct pushes, grant the
> `github-actions[bot]` user bypass rights, or configure a deploy key /
> PAT with elevated permissions and pass it via the `token` secret of
> `actions/checkout`.

## Development environment setup

1. **Install Go 1.26+** (check with `go version`).
2. **Install `git`** and make sure it's on `PATH` (required by `update` and
   by the auto-`.gitignore` feature).
3. **Clone and fetch dependencies:**
   ```powershell
   git clone https://github.com/anushibinj/repo-mapper.git
   cd repo-mapper
   go mod download
   ```
4. **Build:**
   ```powershell
   go build -o bin\repo-mapper.exe .\cmd\repo-mapper
   ```
5. **Run the test suite:**
   ```powershell
   go test ./...
   go test ./... -cover   # with coverage
   ```
6. **Format and vet before committing:**
   ```powershell
   gofmt -l .    # lists any files needing formatting
   gofmt -w .    # applies formatting
   go vet ./...
   ```
7. **Try it against the bundled example fixture** (a small Java/Spring +
   React/Vite app under `examples/billing-app`) or against this repo itself:
   ```powershell
   .\bin\repo-mapper.exe scan -root examples\billing-app
   .\bin\repo-mapper.exe scan -root .
   ```

### Project layout

```
cmd/repo-mapper/     CLI entrypoint and per-command flag/argument handling
internal/
  scanner/           File-tree walking, hashing, ignore-file handling
  parser/            Dispatches files to the plugin(s) that claim them
  plugin/            Plugin interface + static registry
  analyzer/          Turns raw Entities into the canonical Repository model
  generator/         Markdown / Mermaid / JSON renderers
  cache/             .cache/ persistence + automatic .gitignore management
  git/               Thin wrapper around git subprocess calls
  llm/               Optional OpenAI-compatible client for prose summaries
  config/            repo-mapper.yaml loading + defaults
  model/             Canonical data model shared by analyzer/generator
  logger/            Minimal structured logger
plugins/             One package per supported language/framework
  java/ spring/ react/ vite/ node/ sql/ docker/
examples/billing-app/  Example fixture used by integration tests and docs
PRD.md               Full product/design specification
```

### Adding a new plugin

Implement the `plugin.Plugin` interface (see `internal/plugin`), register it
via `init()` in your plugin package, and blank-import it from
`cmd/repo-mapper/main.go` alongside the existing plugins. Plugins only ever
produce `model.Entity` values — turning those into higher-level
components/routes/APIs/tables is solely the analyzer's job.
