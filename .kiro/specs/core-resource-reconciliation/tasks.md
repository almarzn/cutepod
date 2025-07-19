# Implementation Plan

## Phase 1: Core Architecture Refactoring

- [ ] 1. Implement core reconciliation interfaces and abstractions
  - [x] 1.1 Create Resource interface and common types
    - Refactor existing CuteContainer to implement Resource interface
    - Define ResourceType, ResourceReference, and ResourceManager interfaces
    - Create ReconciliationError with proper error classification
    - _Requirements: 6.4, 9.1_ | _Status: Refactor existing_
  - [x] 1.2 Abstract Podman client interactions
    - Extract existing Podman bindings into PodmanClient interface
    - Create adapter pattern for container, network, volume, secret operations
    - Implement mock client for testing and development
    - _Requirements: 1.1, 2.1, 3.1, 4.1_ | _Status: Refactor existing_

  - [x] 1.3 Build manifest registry and object referencing system



    - Refactor chart parsing to use ManifestRegistry pattern
    - Implement name-based cross-resource referencing
    - Remove namespace injection from manifest templates
    - _Requirements: 7.1, 7.2, 9.1_ | _Status: Refactor existing_

## Phase 2: State Comparison and Dependency Engines

- [ ] 2. Implement State Comparison Engine
  - [ ] 2.1 Create StateComparator interface and implementation
    - Extract existing container.Compare logic into generic StateComparator
    - Implement StateDiff structure with create/update/delete operations
    - Add resource fingerprinting for idempotency checks
    - _Requirements: 10.1, 10.3, 10.5_ | _Status: Refactor existing_

  - [ ] 2.2 Build unified change detection system
    - Refactor existing container.GetChanges into generic ResourceManager pattern
    - Standardize Add/Update/Remove/None change types across all resources
    - Implement change execution with proper error handling
    - _Requirements: 10.1, 10.2, 10.3_ | _Status: Refactor existing_

- [ ] 3. Build Dependency Resolution Engine
  - [ ] 3.1 Implement dependency graph builder
    - Create DependencyResolver interface and DependencyGraph structure
    - Build dependency relationships from manifest cross-references
    - Add circular dependency detection and validation
    - _Requirements: 7.1, 7.2, 7.3_ | _Status: New implementation_

  - [ ] 3.2 Implement topological sorting for execution order
    - Create creation and deletion order algorithms
    - Handle parallel execution of independent resources
    - Add dependency failure propagation logic
    - _Requirements: 7.4, 7.5_ | _Status: New implementation_

## Phase 3: Resource Managers and Reconciliation Controller

- [ ] 4. Implement ResourceManager for all resource types
  - [ ] 4.1 Refactor ContainerManager from existing code
    - Extract existing container logic into ResourceManager interface
    - Implement GetDesiredState, GetActualState, CRUD operations
    - Add container-specific dependency resolution (networks, volumes, secrets)
    - _Requirements: 1.1, 1.2, 1.3, 1.4_ | _Status: Refactor existing_

  - [ ] 4.2 Create NetworkManager following ResourceManager pattern
    - Implement CuteNetwork manifest parsing and validation
    - Add Podman network operations through PodmanClient interface
    - Create network state comparison and change detection
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_ | _Status: New implementation_

  - [ ] 4.3 Create VolumeManager following ResourceManager pattern
    - Implement CuteVolume manifest parsing with bind/volume types
    - Add Podman volume operations through PodmanClient interface
    - Create volume state comparison and dependency handling
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_ | _Status: New implementation_

  - [ ] 4.4 Create SecretManager following ResourceManager pattern
    - Implement CuteSecret manifest parsing with base64 handling
    - Add Podman secret operations through PodmanClient interface
    - Create secret state comparison and injection logic
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_ | _Status: New implementation_

- [ ] 5. Build centralized ReconciliationController
  - [ ] 5.1 Create ReconciliationController interface and implementation
    - Refactor existing chart.Upgrade logic into controller pattern
    - Implement full reconciliation workflow: parse → resolve → compare → execute
    - Add comprehensive error handling and recovery mechanisms
    - _Requirements: 6.1, 6.2, 6.3, 6.4_ | _Status: Refactor existing_

  - [ ] 5.2 Implement orphaned resource cleanup
    - Add detection of resources not present in current manifests
    - Create safe orphan removal respecting dependency order
    - Integrate cleanup reporting with existing output system
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_ | _Status: New implementation_

## Phase 4: Enhanced CLI and Output System

- [ ] 6. Implement comprehensive resource labeling strategy
  - [ ] 6.1 Create consistent labeling system
    - Extend existing cutepod.Namespace label to full strategy
    - Add chart name, version, and managed-by labels to all resources
    - Implement label-based resource querying and filtering
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_ | _Status: Refactor existing_

- [ ] 7. Enhance CLI output formatting system
  - [ ] 7.1 Refactor existing lipgloss output into OutputFormatter interface
    - Create TableFormatter, JSONFormatter, YAMLFormatter implementations
    - Enhance existing tree-style output for all resource types
    - Improve status indicators and progress reporting
    - _Requirements: 6.3, 6.4_ | _Status: Refactor existing_

  - [ ] 7.2 Implement comprehensive dry-run and verbose modes
    - Enhance existing dry-run with detailed change previews
    - Add verbose mode with timestamps and Podman command logging
    - Create structured error reporting with actionable suggestions
    - _Requirements: 6.5_ | _Status: Refactor existing_

- [ ] 8. Integrate ReconciliationController with CLI commands
  - [ ] 8.1 Refactor install command to use ReconciliationController
    - Replace existing chart.Install with ReconciliationController.Reconcile
    - Enhance flag handling and output formatting
    - Add proper namespace handling at CLI level only
    - _Requirements: 6.1, 6.3, 6.5_ | _Status: Refactor existing_

  - [ ] 8.2 Refactor upgrade command to use ReconciliationController
    - Replace existing chart.Upgrade with ReconciliationController.Reconcile
    - Enhance state comparison and idempotency logic
    - Improve upgrade-specific output and error handling
    - _Requirements: 6.1, 6.3, 10.1, 10.2_ | _Status: Refactor existing_

## Phase 5: Comprehensive Testing and Error Handling

- [ ] 9. Implement comprehensive error handling and classification
  - [ ] 9.1 Create error classification system
    - Implement ReconciliationError with proper error types
    - Add retry logic for transient Podman API errors
    - Create graceful degradation for partial failures
    - _Requirements: 1.5, 2.5, 4.4, 7.3_ | _Status: Refactor existing_

- [ ] 10. Create comprehensive unit and integration test suite
  - [ ] 10.1 Unit tests for all engines and managers
    - Test StateComparator with various resource configurations
    - Test DependencyResolver with complex dependency scenarios
    - Test all ResourceManagers with mock PodmanClient
    - _Requirements: 10.1, 10.3, 10.5_ | _Status: New implementation_

  - [ ] 10.2 Integration tests for ReconciliationController
    - Test complete reconciliation cycles with real Podman instances
    - Test dependency failure scenarios and recovery
    - Verify idempotency across multiple reconciliation runs
    - _Requirements: 10.1, 10.4, 7.1, 7.2, 7.3_ | _Status: New implementation_

  - [ ] 10.3 CLI and output formatting tests
    - Test all OutputFormatter implementations
    - Test CLI command integration with various scenarios
    - Test error handling and user experience flows
    - _Requirements: 6.3, 6.4, 6.5_ | _Status: New implementation_