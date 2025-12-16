# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Terracotta is a lightweight Terraform module orchestrator written in Go that executes multiple Terraform modules in a dependency-aware order. It reads configuration from YAML files and handles `terraform init`, `plan`, and `apply` steps automatically.

## Build and Development Commands

```bash
# Build the binary (outputs to bin/terracotta)
make build

# Clean build artifacts
make clean

# Run the CLI
./bin/terracotta version
./bin/terracotta plan --config examples/terracotta.yaml
./bin/terracotta apply --config examples/terracotta.yaml --profile <aws-profile>
```

The build process injects version information via ldflags:
- Version: derived from git tags (`git describe --tags --always`)
- Commit: current git commit hash
- Date: build timestamp

## Architecture

### Core Components

The codebase is organized into three main packages:

**1. cmd/** - CLI commands using Cobra framework
- `root.go`: Root command and global flags (`configPath`, `awsProfile`)
- `plan.go`: Runs `terraform init` and `plan` for all modules in dependency order
- `apply.go`: Runs `terraform init` and `apply -auto-approve` for all modules in dependency order
- `version.go`: Displays version information (injected at build time via ldflags)

**2. config/** - Configuration and dependency resolution
- `config.go`: YAML config parsing (defines `Config` and `Module` structs)
- `graph.go`: Dependency graph construction and topological sorting
  - `ExecutionGraph`: Holds module nodes with dependency relationships
  - `BuildExecutionGraph()`: Creates graph from config
  - `TopoSortedModules()`: Performs topological sort to determine execution order
  - Detects cyclic dependencies and validates all dependencies exist

**3. terraform/** - Terraform execution
- `executor.go`: `RunCommand()` wraps terraform CLI execution
  - Sets working directory to module path
  - Captures and prefixes output with module name
  - Returns errors from terraform commands

### Execution Flow

1. Load YAML config from file (`config.LoadConfig`)
2. Build execution graph from config (`config.BuildExecutionGraph`)
3. Perform topological sort to determine module order (`graph.TopoSortedModules`)
4. Set AWS_PROFILE environment variable if `--profile` flag provided
5. For each module in sorted order:
   - Construct full path: `filepath.Join(cfg.BasePath, mod.Path)`
   - Run `terraform init -input=false`
   - Run `terraform plan` (for plan command) or `terraform apply -auto-approve` (for apply command)
6. Display summary with status symbols (✔ success, ✖ failed, ⏭ skipped)

### Key Design Patterns

- **Fail-fast on apply**: Apply stops at first failure and skips remaining modules
- **Continue on plan**: Plan continues through all modules and reports all failures at end
- **Output prefixing**: All terraform output is prefixed with `[module-path]` for clarity
- **Path construction**: Module paths are relative to `base_path` from config (e.g., `environments/dev/shared/network`)

## YAML Configuration Format

```yaml
base_path: environments/dev
modules:
  - path: shared/network
  - path: serviceA/backend
    depends_on: ["shared/network"]
  - path: serviceB/backend
    depends_on: ["shared/network"]
```

- `base_path`: Root directory containing all modules (prepended to each module path)
- `modules`: List of modules with optional `depends_on` array
- Module paths are relative to `base_path`

## Release Process

Releases use GoReleaser triggered by version tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

CI/CD automatically builds and publishes binaries to GitHub Releases.
