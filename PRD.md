# Repo Mapper - Product Requirements Document (PRD)

**Version:** 1.0
**Status:** Draft
**Language:** Go (Golang)
**Target Platform:** Cross-platform CLI (Linux, macOS, Windows)

---

# 1. Vision

Repo Mapper is an AI-first repository analysis tool that continuously generates an up-to-date architectural representation of a codebase.

Unlike traditional architecture tools that produce diagrams for humans, Repo Mapper generates structured documentation specifically optimized for Large Language Models (LLMs) while remaining useful for developers.

Its primary goal is to dramatically reduce AI context pollution by allowing coding assistants to understand repository structure before opening source files.

The generated artifacts become the first source of truth for AI assistants.

---

# 2. Goals

## Functional Goals

* Scan an entire repository.
* Detect project type automatically.
* Understand repository structure.
* Build an internal architecture model.
* Generate Mermaid diagrams.
* Generate AI-friendly JSON indexes.
* Generate Markdown documentation.
* Support incremental updates from Git diffs.
* Execute efficiently in CI/CD.
* Support multiple languages via plugins.

---

## Non-Goals

Repo Mapper is NOT:

* a static code analyzer
* a linter
* a code formatter
* a security scanner
* a documentation website generator

Those concerns belong to other tools.

Repo Mapper only generates architecture knowledge.

---

# 3. Design Principles

## AI First

Everything generated should be optimized for LLM consumption.

Humans are secondary consumers.

---

## Deterministic before AI

Always prefer parsing source code directly.

Only use LLMs for:

* summarisation
* explanations
* architectural descriptions
* naming
* document writing

Never rely on LLMs for extracting information already available through deterministic parsing.

---

## Fast

Designed to run on every Pull Request.

Target:

* <10 seconds for incremental scans
* <60 seconds for full repository scan

---

## Stateless

Every execution should be reproducible.

No database.

Only lightweight cache files.

---

## Plugin Based

Core engine must not know framework specifics.

Framework knowledge belongs in plugins.

---

# 4. High-Level Architecture

```
                Repository

                     │

          Repository Scanner

                     │

          Language Parsers

                     │

        Canonical Repository Model

                     │

     ┌──────────────┼───────────────┐

 Mermaid      Markdown        AI Manifest

 Generator    Generator       Generator

                     │

             Output Directory
```

---

# 5. Project Structure

```
repo-mapper/

    cmd/

    internal/

        scanner/

        parser/

        analyzer/

        model/

        generator/

        llm/

        cache/

        git/

        config/

        plugin/

        logger/

    plugins/

        java/

        spring/

        react/

        vite/

        node/

        docker/

        sql/

    docs/

    examples/

    tests/
```

---

# 6. CLI

```
repo-mapper scan
```

Performs full scan.

---

```
repo-mapper update
```

Uses Git diff.

Only scans changed files.

---

```
repo-mapper clean
```

Deletes cache.

---

```
repo-mapper doctor
```

Validates installation.

---

```
repo-mapper plugins
```

Lists installed plugins.

---

```
repo-mapper graph
```

Outputs architecture graph.

---

```
repo-mapper export
```

Exports JSON model.

---

# 7. Internal Pipeline

```
Repository

↓

Detect Project

↓

Detect Languages

↓

Load Plugins

↓

Parse Files

↓

Build Symbol Graph

↓

Analyze Architecture

↓

Generate Canonical Model

↓

Generate Documentation

↓

Finish
```

---

# 8. Canonical Repository Model

Everything should be represented internally as Go structs.

Example:

```go
type Repository struct {
    Name string

    Languages []Language

    Modules []Module

    Features []Feature

    Components []Component

    Routes []Route

    APIs []API

    Tables []Table
}
```

This is the single source of truth.

Everything else is generated from this.

---

# 9. Scanner

Responsibilities:

* discover files
* ignore binaries
* honour .gitignore
* honour custom ignore file
* compute hashes
* detect changes
* classify files

Should be highly parallel.

---

# 10. Git Module

Responsible for:

* changed files
* staged files
* branch
* commit hash
* merge base
* pull request diff

Must support:

```
repo-mapper update
```

without rescanning entire repository.

---

# 11. Plugin System

Every language/framework implements:

```
type Plugin interface {

    Name() string

    CanParse(file string) bool

    Parse(ctx Context, file string) ([]Entity,error)

}
```

Examples:

* Java Plugin
* Spring Plugin
* React Plugin
* Docker Plugin
* SQL Plugin

## Plugin Loading Strategy (Implementation Note)

Go's native `plugin` package (dynamically loaded `.so`/`.dll` files) does **not**
support Windows and is fragile across Go versions, which conflicts with the
cross-platform requirement in Section "Target Platform".

Instead, plugins are implemented as ordinary Go packages under `plugins/`,
compiled directly into the `repo-mapper` binary, and self-register into a
static, in-memory `plugin.Registry` via an `init()` function:

```go
func init() {
    plugin.Register(New())
}
```

The core engine only ever depends on the `Plugin` interface and the registry —
never on a concrete plugin package — preserving the "core must not know
framework specifics" principle while remaining fully cross-platform and
single-binary. True dynamic/out-of-process plugin loading (e.g. via
subprocess + RPC, similar to HashiCorp's `go-plugin`) is deferred to a later
roadmap phase if third-party plugin distribution becomes a requirement.

## Parsing Strategy (Implementation Note)

Phase 1 plugins (Java, Spring, React, Vite, Node, Docker, SQL) use fast,
deterministic regex/heuristic-based parsing rather than full language ASTs.
This keeps the tool dependency-light and fast, and is sufficient to extract
the structural signals needed (annotations, imports, JSX component usage,
route decorators, etc.). Full AST-based parsing (e.g. via a proper Java
grammar or TypeScript compiler API) is a candidate upgrade in later phases
once accuracy requirements demand it. LLM usage still never substitutes for
this deterministic extraction, per Section 3.

---

# 12. Analyzer

Consumes parsed entities.

Produces semantic knowledge.

Examples:

Java

```
Controller

↓

Service

↓

Repository
```

Spring

```
@RestController

↓

Endpoints
```

React

```
Route

↓

Component

↓

API Calls
```

Database

```
Entity

↓

Table

↓

Relationships
```

---

# 13. Outputs

## Markdown

```
.ai/

    README.md

    backend.md

    frontend.md

    architecture.md

    database.md

    features.md
```

---

## Mermaid

```
.ai/

    diagrams/

        system.mmd

        backend.mmd

        frontend.mmd

        auth.mmd

        database.mmd
```

---

## JSON

```
.ai/

    repo-map.json

    feature-map.json

    api-map.json

    ownership-map.json

    routes.json
```

---

# 14. Mermaid Diagrams

Generate multiple focused diagrams.

Never one giant graph.

Examples:

System

```
Frontend

↓

Backend

↓

Database
```

Backend

```
Controller

↓

Service

↓

Repository
```

Frontend

```
Pages

↓

Components

↓

Hooks

↓

REST API
```

Database

ER Diagram.

Authentication

Flow Diagram.

Feature Ownership

Graph.

---

# 15. AI Manifest

Primary output.

Example:

```json
{
  "feature": "Billing",

  "frontend": [

    "BillingPage.tsx"

  ],

  "backend": [

    "BillingController"

  ],

  "apis":[

    "/billing"

  ],

  "database":[

    "invoice"

  ]
}
```

LLMs should consume this before source code.

---

# 16. Cache

Store

* hashes
* timestamps
* parsed entities
* plugin metadata

Consistent with the "Stateless" principle (Section 3: no database, only
lightweight cache files), the cache is plain JSON on disk — no embedded
database engine.

Example:

```
.cache/

    entities.json

    hashes.json

    files.json
```

Avoid reparsing unchanged files.

---

# 17. Incremental Updates

Workflow:

```
git diff

↓

Changed files

↓

Affected features

↓

Affected diagrams

↓

Affected markdown

↓

Affected JSON

↓

Regenerate only those outputs
```

Never rebuild everything unnecessarily.

---

# 18. LLM Integration

Support any OpenAI-compatible API.

Configuration:

```yaml
llm:

  enabled: true

  endpoint:

  api_key:

  model:
```

Responsibilities:

* write summaries
* improve documentation
* describe architecture
* explain module purpose

LLM must never replace deterministic parsing.

---

# 19. Configuration

```
repo-mapper.yaml
```

Example

```yaml
scan:

  exclude:

    - node_modules

    - build

    - dist

    - target

output:

  directory: .ai

llm:

  enabled: false

plugins:

  java: true

  spring: true

  react: true
```

---

# 20. CI/CD Integration

Example GitHub Actions

```
Checkout

↓

repo-mapper update

↓

Commit updated documentation

↓

Push
```

Should execute automatically after successful builds.

---

# 21. Performance Targets

Repository

10,000 files

Target:

Initial Scan

<60 seconds

Incremental Scan

<10 seconds

Memory

<500 MB

Parallel workers

CPU Count × 2

---

# 22. Logging

Levels

* Debug
* Info
* Warning
* Error

Structured logging only.

---

# 23. Error Handling

Parser failures must not terminate execution.

Example

```
React parser failed

↓

Log warning

↓

Continue scanning
```

Fault tolerance is mandatory.

---

# 24. Future Roadmap

## Phase 1

* Java
* Spring Boot
* React
* Vite
* Mermaid
* Markdown
* JSON
* Git Diff

---

## Phase 2

* TypeScript AST
* Next.js
* Angular
* NestJS
* Docker Compose
* Kubernetes
* Terraform

---

## Phase 3

* C#
* Python
* Go
* Rust
* PHP
* Ruby

---

## Phase 4

* Dependency graph optimisation
* Vector embeddings
* Semantic search
* MCP server integration
* IDE integration
* VS Code extension

---

# 25. Copilot Coding Guidelines

The implementation should follow these architectural rules:

1. Use idiomatic Go and standard project layout.
2. Prefer composition over inheritance.
3. Minimise external dependencies unless they provide significant value.
4. Use interfaces to define plugin contracts.
5. Ensure all modules are independently testable.
6. Use goroutines and worker pools for parallel scanning where beneficial.
7. Never hardcode framework-specific logic in the core engine.
8. Treat the canonical repository model as the single source of truth; all outputs (Markdown, Mermaid, JSON) must be generated from it.
9. Design generators to be deterministic so repeated runs on unchanged repositories produce identical output.
10. Build with future extensibility in mind, including support for additional languages, frameworks, IDE integrations, and AI providers.
11. Prioritise incremental updates and caching to minimise CI execution time.
12. Maintain clear separation between scanning, parsing, analysis, generation, and AI-assisted summarisation.
