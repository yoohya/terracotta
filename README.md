# terracotta

Terracotta is a lightweight Terraform module orchestrator designed to execute multiple modules in a defined order. It reads configuration from a YAML file and handles `terraform init`, `plan`, and `apply` steps automatically per module.

## Features

- Executes Terraform modules in sequence based on a YAML config
- Automatically runs `terraform init` before each plan/apply
- Environment-aware module paths (`/environments/{env}/{service}/{module}`)
- Supports binary and Docker-based execution
- Easily integratable with CI/CD pipelines

## Installation

### Using Prebuilt Binaries

You can download the latest version from [Releases](https://github.com/yoohya/terracotta/releases).

```bash
# Example for macOS amd64
curl -L https://github.com/yoohya/terracotta/releases/download/v0.1.4/terracotta_0.1.4_darwin_amd64.tar.gz | tar -xz terracotta
chmod +x terracotta
./terracotta version
```

### From Source

```bash
git clone https://github.com/yoohya/terracotta.git
cd terracotta
make build
./bin/terracotta version
```

## Usage

### Sample YAML Configuration

```yaml
base_path: environments/dev
modules:
  - path: shared/network
  - path: serviceA/backend
    depends_on:
      - shared/network
  - path: serviceB/backend
    depends_on:
      - shared/network
  - path: serviceC/backend
    depends_on:
      - shared/network
  - path: shared/monitoring
    depends_on:
      - serviceA/backend
      - serviceB/backend
      - serviceC/backend
```

### Execute Plan

```bash
terracotta plan --config examples/terracotta.yaml
```

Available options:
- `--config, -c`: Path to config file (default: `terracotta.yaml`)
- `--profile`: AWS profile to use for authentication
- `--upgrade`: Upgrade providers to the latest version during `terraform init`

Examples:

```bash
# Plan with AWS profile
terracotta plan --config examples/terracotta.yaml --profile my-aws-profile

# Plan with provider upgrade
terracotta plan --config examples/terracotta.yaml --upgrade
```

### Execute Apply

```bash
terracotta apply --config examples/terracotta.yaml
```

Available options:
- `--config, -c`: Path to config file (default: `terracotta.yaml`)
- `--profile`: AWS profile to use for authentication
- `--upgrade`: Upgrade providers to the latest version during `terraform init`

Examples:

```bash
# Apply with AWS profile
terracotta apply --config examples/terracotta.yaml --profile my-aws-profile

# Apply with provider upgrade
terracotta apply --config examples/terracotta.yaml --upgrade
```

### Show Version

```bash
terracotta version
```

## Development

### Running Tests

```bash
# Run all tests (cache disabled with -count=1)
make test

# Run tests with coverage report
make test-coverage

# Run tests with race detector
make test-race

# Run all quality checks
make test-all

# Run linter
make lint

# Install golangci-lint (if not already installed)
make install-lint

# Clean test cache and coverage files
make clean-all
```

> **Note**: All test targets use `-count=1` to disable Go's test cache, ensuring fresh test execution every time. This is important for tests that depend on external resources like files or environment variables.

### Test Coverage

Current test coverage:
- `config` package: 100%
- `terraform` package: 70%

The test suite includes:
- Unit tests for YAML configuration parsing
- Dependency graph construction and validation
- Topological sort algorithm testing
- Cyclic dependency detection
- Unknown dependency error handling
- Terraform command execution (integration tests)

**Note**: Integration tests that execute actual Terraform commands require Terraform to be installed. In CI environments, Terraform is automatically installed via `hashicorp/setup-terraform` action. Locally, if Terraform is not found, these tests are gracefully skipped.

### Building

```bash
# Build the binary
make build

# Clean build artifacts
make clean
```

## Release

Releases are triggered by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

CI/CD will automatically build and publish binaries to GitHub Releases.

## License

MIT