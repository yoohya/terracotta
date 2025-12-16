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

## Release

Releases are triggered by pushing a version tag:

```bash
git tag v0.1.0
git push origin v0.1.0
```

CI/CD will automatically build and publish binaries to GitHub Releases.

## License

MIT