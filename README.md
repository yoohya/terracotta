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
curl -L https://github.com/yoohya/terracotta/releases/download/v0.1.0/terracotta_0.1.0_darwin_amd64.tar.gz | tar -xz
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
  - name: network
    service: shared
  - name: backend
    service: serviceA
  - name: backend
    service: serviceB
  - name: backend
    service: serviceC
  - name: monitoring
    service: shared
```

### Execute Plan

```bash
terracotta plan --config examples/terracotta.yaml
```

### Execute Apply

```bash
terracotta apply --config examples/terracotta.yaml
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