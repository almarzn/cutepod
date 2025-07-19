# Project Structure

## Root Directory Layout
```
cutepod/
├── main.go                 # CLI entrypoint
├── go.mod/go.sum          # Go module definition
├── Makefile               # Build automation
├── README.md              # Project documentation
├── cmd/cli/               # CLI command implementations
├── internal/              # Internal packages (not importable)
├── examples/              # Example charts and demos
├── schemas/               # Resource validation schemas
├── e2e/                   # End-to-end tests
├── bin/                   # Build output (gitignored)
└── .codex/                # Autonomous agent workspace
```

## Package Organization

### CLI Layer (`cmd/cli/`)
- `root.go` - Main CLI setup and Execute() function
- `install.go` - Install command implementation
- `upgrade.go` - Upgrade command implementation  
- `lint.go` - Validation command implementation
- `reinit.go` - Restart recovery command

### Internal Packages (`internal/`)
- `chart/` - Chart parsing, templating, and validation
- `container/` - Container resource management and Podman operations
- `podman/` - Podman client adapter and bindings
- `resource/` - Resource type definitions and interfaces
- `object/` - Object change detection and reconciliation
- `meta/` - Metadata parsing and base types

### Chart Structure (Examples)
```
chart/
├── Chart.yaml             # Chart metadata (name, version)
├── values.yaml            # Default template values
└── templates/             # Go template files
    ├── container.yaml     # CuteContainer definitions
    ├── secret.yaml        # CuteSecret definitions
    └── network.yaml       # CuteNetwork definitions
```

## Coding Conventions

### File Naming
- Use snake_case for files: `container_test.go`, `parse_options.go`
- Test files: `*_test.go` alongside source files
- Integration tests: `*_integration_test.go`

### Package Structure
- Each `internal/` package should have clear responsibility
- Interfaces defined in separate files when complex
- Mock implementations for testing: `mock.go`, `mock_test.go`

### Resource Types
- All resources implement common `Resource` interface
- Use Kubernetes-style metadata (`metav1.ObjectMeta`)
- Resource kinds: `CuteContainer`, `CutePod`, `CuteNetwork`, `CuteVolume`, `CuteSecret`, `CuteSecretStore`, `CuteExtension`

### Error Handling
- Wrap errors with context: `fmt.Errorf("failed to parse %s: %w", path, err)`
- Use structured error messages for user-facing errors
- Validation errors should include file path and line context

### Testing Patterns
- Unit tests alongside source files
- Integration tests in separate `*_integration_test.go` files
- E2E tests in dedicated `e2e/` directory with real Podman containers
- Mock Podman client for unit testing container operations