# Core Resource Reconciliation Design

## Overview

The core resource reconciliation system implements a declarative infrastructure management engine for Cutepod that manages Podman containers, networks, volumes, and secrets through YAML manifests. The system follows a controller pattern where it continuously compares desired state (defined in CuteContainer, CuteNetwork, CuteVolume, CuteSecret, and CutePod manifests) with actual state (existing Podman resources) and applies necessary changes to achieve convergence.

The reconciliation engine is designed to be idempotent, dependency-aware, and provides comprehensive status reporting with clean, readable CLI output. It uses consistent labeling for resource tracking and handles orphaned resource cleanup automatically. Namespace is provided only at command execution time, not embedded in manifests.

## Architecture

The reconciliation system follows a layered architecture with clear separation of concerns:

```
┌─────────────────────────────────────────────────────────────┐
│                    Reconciliation Controller                │
├─────────────────────────────────────────────────────────────┤
│  Resource Managers (Container, Network, Volume, Secret)    │
├─────────────────────────────────────────────────────────────┤
│              Dependency Resolution Engine                   │
├─────────────────────────────────────────────────────────────┤
│                 State Comparison Engine                     │
├─────────────────────────────────────────────────────────────┤
│                   Podman Client Adapter                     │
└─────────────────────────────────────────────────────────────┘
```

**Design Rationale:** The layered approach ensures modularity and testability. Each layer has a single responsibility, making the system easier to maintain and extend. The controller pattern provides a familiar paradigm for Kubernetes-experienced developers.

## Components and Interfaces

### Reconciliation Controller

The main orchestrator that coordinates the reconciliation process:

```go
type ReconciliationController interface {
    Reconcile(ctx context.Context, manifests []Manifest, chartName string, dryRun bool) (*ReconciliationResult, error)
    GetStatus(chartName string) (*ReconciliationStatus, error)
}

type ReconciliationResult struct {
    CreatedResources []ResourceAction
    UpdatedResources []ResourceAction
    DeletedResources []ResourceAction
    Errors          []error
    Summary         string
}
```

**Design Rationale:** The controller interface is simple and focused, accepting manifests and returning detailed results. The context parameter enables cancellation and timeout handling.

### Resource Managers

Each resource type has a dedicated manager implementing a common interface:

```go
type ResourceManager interface {
    GetDesiredState(manifests []Manifest) ([]Resource, error)
    GetActualState(ctx context.Context, chartName string) ([]Resource, error)
    CreateResource(ctx context.Context, resource Resource) error
    UpdateResource(ctx context.Context, desired, actual Resource) error
    DeleteResource(ctx context.Context, resource Resource) error
    CompareResources(desired, actual Resource) (bool, error)
}

// Specific implementations
type ContainerManager struct{}
type NetworkManager struct{}
type VolumeManager struct{}
type SecretManager struct{}
type PodManager struct{}
```

**Design Rationale:** The common interface ensures consistent behavior across resource types while allowing type-specific implementations. This pattern makes it easy to add new resource types in the future.

### Dependency Resolution Engine

Manages resource creation order and dependency tracking:

```go
type DependencyResolver interface {
    BuildDependencyGraph(resources []Resource) (*DependencyGraph, error)
    GetCreationOrder(graph *DependencyGraph) ([][]Resource, error)
    GetDeletionOrder(graph *DependencyGraph) ([][]Resource, error)
}

type DependencyGraph struct {
    Nodes map[string]*ResourceNode
    Edges map[string][]string
}
```

**Design Rationale:** Explicit dependency modeling prevents race conditions and ensures resources are created in the correct order. The graph structure allows for parallel processing of independent resources within each dependency level.

### State Comparison Engine

Handles the core logic of comparing desired vs actual state:

```go
type StateComparator interface {
    CompareStates(desired, actual []Resource) (*StateDiff, error)
    ShouldUpdate(desired, actual Resource) (bool, []string, error)
}

type StateDiff struct {
    ToCreate []Resource
    ToUpdate []ResourcePair
    ToDelete []Resource
    Unchanged []Resource
}
```

**Design Rationale:** Centralized comparison logic ensures consistent behavior across all resource types. The detailed diff structure provides transparency into what changes will be applied.

### Podman Client Adapter

Abstracts Podman API interactions:

```go
type PodmanClient interface {
    // Container operations
    CreateContainer(ctx context.Context, spec ContainerSpec) (*Container, error)
    UpdateContainer(ctx context.Context, id string, spec ContainerSpec) error
    DeleteContainer(ctx context.Context, id string, force bool) error
    ListContainers(ctx context.Context, filters map[string]string) ([]Container, error)
    
    // Network operations
    CreateNetwork(ctx context.Context, spec NetworkSpec) (*Network, error)
    DeleteNetwork(ctx context.Context, id string) error
    ListNetworks(ctx context.Context, filters map[string]string) ([]Network, error)
    
    // Volume operations
    CreateVolume(ctx context.Context, spec VolumeSpec) (*Volume, error)
    DeleteVolume(ctx context.Context, name string) error
    ListVolumes(ctx context.Context, filters map[string]string) ([]Volume, error)
    
    // Secret operations
    CreateSecret(ctx context.Context, spec SecretSpec) (*Secret, error)
    UpdateSecret(ctx context.Context, name string, spec SecretSpec) error
    DeleteSecret(ctx context.Context, name string) error
    ListSecrets(ctx context.Context, filters map[string]string) ([]Secret, error)
}
```

**Design Rationale:** The adapter pattern isolates Podman-specific code and makes the system testable with mock implementations. The interface closely mirrors Podman's API while providing necessary abstractions.

## Manifest Parsing and Object Registry

### Manifest Structure

Cutepod uses Kubernetes-style YAML manifests without namespace fields. Objects are referenced by name within the same chart context:

```yaml
apiVersion: cutepod/v1alpha0
kind: CuteContainer
metadata:
  name: web-server
spec:
  image: nginx:latest
  ports:
    - containerPort: 80
      hostPort: 8080
  volumes:
    - name: web-content  # References CuteVolume by name
      mountPath: /usr/share/nginx/html
  networks:
    - web-network        # References CuteNetwork by name
  secrets:
    - name: db-credentials  # References CuteSecret by name
      env: true
---
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: web-network
spec:
  driver: bridge
  subnet: "172.20.0.0/16"
```

### Object Registry and Referencing

Objects are referenced by name within the chart context. The system maintains an internal registry during parsing:

```go
type ManifestRegistry struct {
    Resources map[string]Resource
    Dependencies map[string][]string
}

type ObjectReference struct {
    Name string
    Type ResourceType
}

// Objects reference each other by name only
type VolumeMount struct {
    Name      string  // Volume name reference
    MountPath string
    ReadOnly  bool
}

type SecretReference struct {
    Name string  // Secret name reference
    Env  bool    // Mount as environment variables
    Path string  // Mount as file (optional)
}
```

**Design Rationale:** Name-based referencing keeps manifests clean and portable. The registry pattern enables validation of references during parsing and dependency resolution.

## Data Models

### Core Resource Types

```go
type Resource interface {
    GetType() ResourceType
    GetName() string
    GetLabels() map[string]string
    GetDependencies() []ResourceReference
}

type ResourceType string
const (
    ResourceTypeContainer ResourceType = "container"
    ResourceTypeNetwork   ResourceType = "network"
    ResourceTypeVolume    ResourceType = "volume"
    ResourceTypeSecret    ResourceType = "secret"
    ResourceTypePod       ResourceType = "pod"
)

type ResourceReference struct {
    Type ResourceType
    Name string
}
```

### Container Resource

```go
type ContainerResource struct {
    Name         string
    Image        string
    Ports        []PortMapping
    Environment  map[string]string
    Volumes      []VolumeMount
    Networks     []string
    Secrets      []SecretReference
    Resources    ResourceLimits
    RestartPolicy string
    Labels       map[string]string
}

type ResourceLimits struct {
    CPULimit    string
    MemoryLimit string
}
```

### Network Resource

```go
type NetworkResource struct {
    Name      string
    Driver    string
    Options   map[string]string
    Subnet    string
    Labels    map[string]string
}
```

### Volume Resource

```go
type VolumeResource struct {
    Name      string
    Type      VolumeType
    Driver    string
    Options   map[string]string
    HostPath  string // for bind mounts
    Labels    map[string]string
}

type VolumeType string
const (
    VolumeTypeBind   VolumeType = "bind"
    VolumeTypeVolume VolumeType = "volume"
)
```

### Secret Resource

```go
type SecretResource struct {
    Name      string
    Data      map[string][]byte
    Type      SecretType
    Labels    map[string]string
}

type SecretType string
const (
    SecretTypeOpaque SecretType = "opaque"
)
```

**Design Rationale:** The resource models closely mirror the YAML manifest structure while providing type safety. The common Resource interface enables generic processing while specific types handle their unique attributes.

## Error Handling

### Error Classification

```go
type ReconciliationError struct {
    Type        ErrorType
    Resource    ResourceReference
    Message     string
    Cause       error
    Recoverable bool
}

type ErrorType string
const (
    ErrorTypeDependency    ErrorType = "dependency"
    ErrorTypeValidation    ErrorType = "validation"
    ErrorTypePodmanAPI     ErrorType = "podman_api"
    ErrorTypeConfiguration ErrorType = "configuration"
)
```

### Error Handling Strategy

1. **Dependency Errors**: When a dependency fails, halt creation of dependent resources but continue with independent resources
2. **Validation Errors**: Report immediately and skip the invalid resource
3. **Podman API Errors**: Retry with exponential backoff for transient errors, fail fast for permanent errors
4. **Configuration Errors**: Report and halt reconciliation to prevent inconsistent state

**Design Rationale:** Categorized errors enable appropriate handling strategies. Continuing with independent resources maximizes the success rate of each reconciliation cycle.

## Testing Strategy

### Unit Testing

- **Resource Managers**: Test each manager in isolation with mock Podman client
- **Dependency Resolution**: Test graph building and ordering with various dependency scenarios
- **State Comparison**: Test comparison logic with different resource configurations
- **Error Handling**: Test error scenarios and recovery mechanisms

### Integration Testing

- **End-to-End Reconciliation**: Test complete reconciliation cycles with real Podman instances
- **Dependency Scenarios**: Test complex dependency chains and failure scenarios
- **Idempotency**: Verify multiple reconciliation runs produce consistent results
- **Orphan Cleanup**: Test detection and removal of orphaned resources

### Test Data Strategy

```go
type TestScenario struct {
    Name              string
    InitialState      []Resource
    DesiredManifests  []Manifest
    ExpectedActions   []ResourceAction
    ExpectedErrors    []error
}
```

**Design Rationale:** Scenario-based testing ensures comprehensive coverage of real-world use cases. The test data structure makes it easy to add new test cases and maintain existing ones.

### Reconciliation Flow

The reconciliation process follows this sequence:

1. **Parse Manifests**: Convert YAML manifests to internal resource representations
2. **Build Dependency Graph**: Analyze resource dependencies and create execution order
3. **Get Current State**: Query Podman for existing resources in the namespace
4. **Compare States**: Identify resources to create, update, or delete
5. **Execute Changes**: Apply changes in dependency order, handling errors appropriately
6. **Cleanup Orphans**: Remove resources no longer defined in manifests
7. **Report Results**: Provide detailed summary of all actions taken

**Design Rationale:** The sequential flow ensures consistency and provides clear checkpoints for error handling and status reporting.

### Labeling Strategy

All managed resources receive consistent labels:

```go
const (
    LabelChart     = "cutepod.io/chart"
    LabelVersion   = "cutepod.io/version"
    LabelManagedBy = "cutepod.io/managed-by"
)
```

**Design Rationale:** Consistent labeling enables reliable resource tracking and filtering. The namespace-based approach allows multiple Cutepod instances to coexist without interference.

## CLI Design and Output Formatting

### Command Structure

The CLI follows a clean, intuitive structure with namespace provided at execution time:

```bash
# Install resources from a chart
cutepod install <chart-path> [flags]

# Upgrade/reconcile existing resources
cutepod upgrade <chart-path> [flags]

# Common flags
--dry-run          Preview changes without applying them
--verbose, -v      Show detailed operation logs
--output, -o       Output format (table, json, yaml)
```

### Output Design Philosophy

The CLI output prioritizes clarity and actionability:

- **Clean Visual Hierarchy**: Use consistent spacing and alignment
- **Minimal Color Usage**: Strategic use of color for status indication
- **Progress Indication**: Clear progress for long-running operations
- **Error Context**: Detailed error messages with actionable suggestions
- **Summary Focus**: Concise summaries with option for detailed output

### Standard Output Format

#### Installation/Upgrade Output

```
Installing chart: ./my-app]

┌─────────────────────────────────────────────────────────────┐
│ Reconciling Resources                                       │
└─────────────────────────────────────────────────────────────┘

Networks:
  ✓ web-network        created
  ✓ db-network         created

Volumes:
  ✓ web-content        created
  ✓ db-data           created

Secrets:
  ✓ db-credentials     created
  ✓ api-keys          created

Containers:
  ✓ web-server         created
  ✓ database          created
  ✓ cache             updated

┌─────────────────────────────────────────────────────────────┐
│ Summary                                                     │
└─────────────────────────────────────────────────────────────┘

Resources: 7 created, 1 updated, 0 deleted
Duration: 12.3s
Status: Success
```

#### Dry-Run Output

```
Preview: ./my-app → chart-name

┌─────────────────────────────────────────────────────────────┐
│ Planned Changes                                             │
└─────────────────────────────────────────────────────────────┘

Networks:
  + web-network        will be created
  + db-network         will be created

Containers:
  + web-server         will be created
  ~ database          will be updated
    - image: postgres:13 → postgres:14
    - memory: 512Mi → 1Gi

┌─────────────────────────────────────────────────────────────┐
│ Summary                                                     │
└─────────────────────────────────────────────────────────────┘

Planned: 6 create, 1 update, 0 delete
Run without --dry-run to apply changes
```

#### Error Output

```
Installing chart: ./my-app
Chart name: development

┌─────────────────────────────────────────────────────────────┐
│ Reconciling Resources                                       │
└─────────────────────────────────────────────────────────────┘

Networks:
  ✓ web-network        created

Containers:
  ✗ web-server         failed
    Error: image 'nginx:invalid' not found
    Suggestion: Check image name and tag

┌─────────────────────────────────────────────────────────────┐
│ Summary                                                     │
└─────────────────────────────────────────────────────────────┘

Resources: 1 created, 0 updated, 0 deleted
Errors: 1 failed
Status: Failed

Use --verbose for detailed error information
```

#### Verbose Output

```bash
cutepod install development ./my-app --verbose
```

```
Installing chart: ./my-app
Chart name: test
Chart version: 1.2.3

┌─────────────────────────────────────────────────────────────┐
│ Parsing Manifests                                           │
└─────────────────────────────────────────────────────────────┘

Found 8 resources:
  - 2 CuteNetwork
  - 2 CuteVolume  
  - 2 CuteSecret
  - 2 CuteContainer

┌─────────────────────────────────────────────────────────────┐
│ Dependency Analysis                                         │
└─────────────────────────────────────────────────────────────┘

Execution order:
  1. Networks: web-network, db-network
  2. Volumes: web-content, db-data
  3. Secrets: db-credentials, api-keys
  4. Containers: web-server, database

┌─────────────────────────────────────────────────────────────┐
│ Reconciling Resources                                       │
└─────────────────────────────────────────────────────────────┘

[12:34:56] Creating network web-network
[12:34:56] → podman network create web-network --driver bridge
[12:34:57] ✓ Network web-network created (1.2s)

[12:34:57] Creating container web-server
[12:34:57] → podman run -d --name web-server nginx:latest
[12:34:59] ✓ Container web-server created (2.1s)

┌─────────────────────────────────────────────────────────────┐
│ Summary                                                     │
└─────────────────────────────────────────────────────────────┘

Resources: 8 created, 0 updated, 0 deleted
Duration: 8.7s
Status: Success
```

### Status Indicators

```go
type StatusIndicator string
const (
    StatusSuccess   StatusIndicator = "✓"  // Green
    StatusFailed    StatusIndicator = "✗"  // Red  
    StatusUpdated   StatusIndicator = "~"  // Yellow
    StatusCreated   StatusIndicator = "+"  // Green
    StatusDeleted   StatusIndicator = "-"  // Red
    StatusProgress  StatusIndicator = "→"  // Blue
)
```

### Output Formatter Interface

```go
type OutputFormatter interface {
    FormatReconciliationStart(chartName, chartPath string)
    FormatResourceAction(action ResourceAction)
    FormatSummary(result ReconciliationResult)
    FormatError(err ReconciliationError)
    FormatDryRun(diff StateDiff)
}

type TableFormatter struct {
    Verbose bool
    NoColor bool
}

type JSONFormatter struct{}
type YAMLFormatter struct{}
```

**Design Rationale:** The clean, structured output provides immediate visual feedback while maintaining professional appearance. The consistent use of symbols and spacing creates a familiar, tool-like experience similar to kubectl or docker CLI tools.