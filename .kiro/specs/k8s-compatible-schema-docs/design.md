# Design Document: Kubernetes-Compatible Schema Documentation (MVP)

## Overview

This MVP will generate Kubernetes Custom Resource Definition (CRD) files for all Cutepod resource types using controller-gen, enabling basic validation and documentation. The focus is on getting working CRDs with basic validation markers and simple documentation of what's implemented and what needs testing.

The design provides a foundation for Kubernetes tooling compatibility while keeping the initial implementation simple and focused on core functionality.

## Architecture

### Code Generation Pipeline

The CRD generation system uses controller-gen (the standard Kubernetes tooling) for CRD generation, integrated with custom documentation generation:

```
Go Resource Types → controller-gen → CRD YAML Files → Documentation Generator → Markdown Docs
     (source)      (with markers)      (output)         (processing)         (final docs)
```

**Design Decision**: Using controller-gen ensures compatibility with Kubernetes ecosystem standards and provides robust OpenAPI v3 schema generation. This leverages the same tooling used by Kubernetes controllers and operators, ensuring high-quality CRD output that's compatible with all Kubernetes tooling.

### Directory Structure

```
cutepod/
├── crds/                           # Generated CRD files
│   ├── cutecontainer-crd.yaml
│   ├── cutenetwork-crd.yaml
│   ├── cutevolume-crd.yaml
│   ├── cutesecret-crd.yaml
│   └── cutepod-crd.yaml
├── docs/
│   └── feature-status.md           # Simple status documentation
└── cmd/cli/
    └── generate.go                 # CLI command for generation
```

**Design Decision**: Minimal directory structure focuses on essential CRD generation with basic documentation.

## Components and Interfaces

### 1. Controller-gen Integration

Uses controller-gen with kubebuilder markers to automatically generate CRDs from Go struct definitions:

```go
// Example resource with controller-gen markers
// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Namespaced,shortName=cc
// +kubebuilder:subresource:status
type ContainerResource struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   CuteContainerSpec   `json:"spec,omitempty"`
    Status CuteContainerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:generate=true
type CuteContainerSpec struct {
    // +kubebuilder:validation:Required
    // +kubebuilder:validation:MinLength=1
    Image string `json:"image"`
    
    // +kubebuilder:validation:Optional
    Command []string `json:"command,omitempty"`
    
    // +kubebuilder:validation:Minimum=1
    // +kubebuilder:validation:Maximum=65535
    Ports []ContainerPort `json:"ports,omitempty"`
}
```

**Controller-gen Markers Used**:
- `+kubebuilder:object:root=true` - Marks the type as a Kubernetes resource
- `+kubebuilder:resource` - Configures resource metadata (scope, shortNames, etc.)
- `+kubebuilder:validation:*` - Adds OpenAPI validation constraints
- `+kubebuilder:subresource:status` - Enables status subresource
- `+groupName=cutepod` - Sets the API group name

**Build Integration**:
```makefile
# Makefile integration
generate:
	controller-gen crd:crdVersions=v1 paths=./internal/resource/... output:crd:artifacts:config=crds/
	controller-gen object paths=./internal/resource/...
```

**Design Decision**: Controller-gen is the standard tool used by Kubernetes operators and controllers, ensuring our CRDs are fully compatible with the Kubernetes ecosystem and follow best practices.

### 2. Basic Documentation Generator

Creates simple feature status documentation:

```go
type DocGenerator struct {
    resourceTypes []string
}

type FeatureStatus struct {
    ResourceType    string
    CRDGenerated    bool
    ValidationAdded bool
    TestingNeeded   bool
}
```

**Output Features**:
- Simple feature status tracking
- List of generated CRDs
- Documentation of what's implemented vs what needs testing

## Data Models

### Core Resource Types

Based on the existing codebase analysis, the system will generate CRDs for these resource types:

1. **CuteContainer** (`cutepod/v1alpha1`)
   - Primary workload resource with comprehensive container specification
   - Includes ports, volumes, environment, health checks, security context
   - Dependencies: CuteNetwork, CuteVolume, CuteSecret, CutePod

2. **CuteNetwork** (`cutepod/v1alpha1`)
   - Network isolation and connectivity configuration
   - Supports custom drivers and subnet configuration
   - No dependencies (foundational resource)

3. **CuteVolume** (`cutepod/v1alpha1`)
   - Storage abstraction with multiple volume types
   - Supports hostPath, emptyDir, and named volume types
   - Enhanced with security context and ownership controls
   - No dependencies (foundational resource)

4. **CuteSecret** (`cutepod/v1alpha1`)
   - Sensitive data management with base64 encoding
   - Supports environment variable and file mounting
   - No dependencies (foundational resource)

5. **CutePod** (`cutepod/v1alpha1`)
   - Container grouping and shared lifecycle management
   - Dependencies: CuteContainer references

6. **CuteSecretStore** (`cutepod/v1alpha1`) - Future Extension
   - External secret provider integration
   - Dependencies: External secret management systems

7. **CuteExtension** (`cutepod/v1alpha1`) - Future Extension
   - Custom resource transformation and extension points
   - Dependencies: Configurable based on extension type

### OpenAPI Schema Structure

Each CRD follows this consistent structure:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: [plural].[group]
spec:
  group: cutepod
  scope: Namespaced
  names:
    plural: [resource-type]s
    singular: [resource-type]
    kind: Cute[ResourceType]
    shortNames: [abbreviated-forms]
  versions:
  - name: v1alpha1
    served: true
    storage: true
    schema:
      openAPIV3Schema:
        type: object
        required: ["apiVersion", "kind", "metadata", "spec"]
        properties:
          apiVersion:
            type: string
            enum: ["cutepod/v1alpha1"]
          kind:
            type: string
            enum: ["Cute[ResourceType]"]
          metadata:
            type: object
            # Standard Kubernetes metadata
          spec:
            type: object
            # Resource-specific schema
```

**Design Decision**: Using `cutepod` as the API group maintains clear separation from Kubernetes core resources while following established conventions.

## Error Handling

### Schema Generation Errors

```go
type SchemaError struct {
    ResourceType string
    Field        string
    Message      string
    Cause        error
}

type ValidationError struct {
    CRDPath     string
    JSONPath    string
    Message     string
    Suggestion  string
}
```

**Error Categories**:
1. **Type Analysis Errors**: Issues parsing Go struct definitions
2. **Schema Validation Errors**: Invalid OpenAPI v3 schema generation
3. **CRD Validation Errors**: Non-compliant Kubernetes CRD structure
4. **File Generation Errors**: I/O issues during file creation

**Error Recovery**:
- Continue processing other resource types if one fails
- Provide detailed error messages with file paths and line numbers
- Suggest fixes for common validation issues
- Fail fast on critical errors that affect all resources

### Lint Integration Errors

The `cutepod lint` command will integrate CRD validation:

```go
type LintResult struct {
    Valid      bool
    Errors     []ValidationError
    Warnings   []ValidationWarning
    FilePath   string
    LineNumber int
}
```

**Validation Process**:
1. Load appropriate CRD schema for resource type
2. Validate YAML structure against OpenAPI schema
3. Check cross-resource dependencies
4. Verify template rendering with CRD constraints

## Testing Strategy

### Unit Testing

**Schema Extraction Tests**:
```go
func TestSchemaExtractor_ExtractResourceSchema(t *testing.T) {
    extractor := NewSchemaExtractor()
    schema, err := extractor.ExtractResourceSchema(reflect.TypeOf(ContainerResource{}))
    
    assert.NoError(t, err)
    assert.Equal(t, "CuteContainer", schema.Kind)
    assert.NotNil(t, schema.Schema.Properties["spec"])
}
```

**CRD Generation Tests**:
- Validate generated CRDs against Kubernetes OpenAPI specification
- Test schema completeness for all resource types
- Verify validation constraint conversion accuracy

**Documentation Generation Tests**:
- Ensure all fields are documented
- Validate example YAML syntax and completeness
- Test cross-reference link generation

### Integration Testing

**Build Integration Tests**:
```go
func TestMakeGenerate_UpdatesAllCRDs(t *testing.T) {
    // Modify a Go resource type
    // Run make generate
    // Verify CRD files are updated
    // Verify documentation is regenerated
}
```

**Lint Integration Tests**:
```go
func TestCutepodLint_ValidatesAgainstCRDs(t *testing.T) {
    // Create invalid resource YAML
    // Run cutepod lint
    // Verify specific validation errors are reported
}
```

### End-to-End Testing

**Tooling Integration Tests**:
- Test CRD loading in kubectl (if available)
- Validate IDE schema support with generated CRDs
- Test documentation generation pipeline
- Verify backward compatibility with existing charts

**Performance Tests**:
- Measure CRD generation time for all resource types
- Test lint performance with CRD validation enabled
- Validate memory usage during schema processing

### Test Data Management

**Golden File Testing**:
- Maintain reference CRD files for regression testing
- Use golden file comparison for documentation output
- Version control test fixtures for different resource configurations

**Example Validation**:
- Automatically validate all documentation examples
- Test examples against generated CRDs
- Ensure examples demonstrate key features and edge cases

## Implementation Phases

### Phase 1: Core Infrastructure
- Add controller-gen markers to existing Go resource types
- Configure controller-gen for CRD generation
- Generate CRDs for existing resource types (Container, Network, Volume, Secret, Pod)
- Integrate with build system (`make generate` command)

### Phase 2: Validation Integration
- Extract validation rules from existing `Validate()` methods
- Convert Go validation logic to OpenAPI constraints
- Integrate CRD validation into `cutepod lint` command
- Add comprehensive error reporting with file paths and line numbers

### Phase 3: Documentation Generation
- Implement DocGenerator with markdown output
- Create comprehensive field documentation
- Generate working examples for each resource type
- Add feature status tracking and cross-references

### Phase 4: Advanced Features
- Support for future extension types (CuteSecretStore, CuteExtension)
- API versioning support for backward compatibility
- Enhanced IDE integration testing
- Performance optimization and caching

**Design Decision**: Phased approach allows incremental delivery of value while building on solid foundations, enabling early feedback and course correction.