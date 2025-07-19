# Technology Stack

## Language & Runtime
- **Go 1.24+** - Primary language with toolchain go1.24.4
- **Podman** - Container runtime (libpod via containers/libpod)
- Linux/macOS support with rootless Podman

## Key Dependencies
- **CLI Framework**: Cobra (github.com/spf13/cobra) - idiomatic Go CLI development
- **Templating**: Go templates with Sprig functions (github.com/Masterminds/sprig/v3)
- **YAML Processing**: goccy/go-yaml for parsing and validation
- **Podman Integration**: containers/podman/v5 bindings for container operations
- **Kubernetes Types**: k8s.io/apimachinery for metadata structures
- **UI/UX**: charmbracelet/lipgloss for terminal styling, briandowns/spinner for progress

## Build System & Commands

### Core Commands
```bash
# Build the CLI binary
make build

# Run end-to-end tests
make e2e

# Clean build artifacts
make clean
```

### Development Workflow
```bash
# Build and test locally
go build -o bin/cutepod ./main.go

# Run tests
go test ./...

# Run specific package tests
go test ./internal/chart -v
```

### CLI Usage Patterns
```bash
# Validate chart templates and YAML
cutepod lint <path-to-chart>

# Install containers (preview with --dry-run)
cutepod install <namespace> <chart> [--dry-run] [--verbose]

# Reconcile and update containers
cutepod upgrade <namespace> <chart> [--dry-run] [--verbose]

# Restart containers after system restart
cutepod reinit [namespace]
```

## Architecture Patterns
- **Adapter Pattern**: Podman client abstraction in `internal/podman`
- **Command Pattern**: CLI commands in `cmd/cli` with Cobra
- **Template Pattern**: Go templates with Helm-like chart structure
- **Resource Management**: Kubernetes-inspired resource types and reconciliation