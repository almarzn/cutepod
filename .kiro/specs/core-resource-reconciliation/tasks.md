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

- [x] 2. Implement State Comparison Engine
  - [x] 2.1 Create StateComparator interface and implementation
    - Extract existing container.Compare logic into generic StateComparator
    - Implement StateDiff structure with create/update/delete operations
    - Add resource fingerprinting for idempotency checks
    - _Requirements: 10.1, 10.3, 10.5_ | _Status: Refactor existing_

  - [x] 2.2 Build unified change detection system
    - Refactor existing container.GetChanges into generic ResourceManager pattern
    - Standardize Add/Update/Remove/None change types across all resources
    - Implement change execution with proper error handling
    - _Requirements: 10.1, 10.2, 10.3_ | _Status: Refactor existing_

- [x] 3. Build Dependency Resolution Engine
  - [x] 3.1 Implement dependency graph builder
    - Create DependencyResolver interface and DependencyGraph structure
    - Build dependency relationships from manifest cross-references
    - Add circular dependency detection and validation
    - _Requirements: 7.1, 7.2, 7.3_ | _Status: New implementation_

  - [x] 3.2 Implement topological sorting for execution order
    - Create creation and deletion order algorithms
    - Handle parallel execution of independent resources
    - Add dependency failure propagation logic
    - _Requirements: 7.4, 7.5_ | _Status: New implementation_

## Phase 3: Resource Managers and Reconciliation Controller

- [ ] 4. Implement ResourceManager for all resource types
  - [x] 4.1 Refactor ContainerManager from existing code
    - Extract existing container logic into ResourceManager interface
    - Implement GetDesiredState, GetActualState, CRUD operations
    - Add container-specific dependency resolution (networks, volumes, secrets)
    - _Requirements: 1.1, 1.2, 1.3, 1.4_ | _Status: Refactor existing_

  - [x] 4.2 Create NetworkManager following ResourceManager pattern
    - Implement CuteNetwork manifest parsing and validation
    - Add Podman network operations through PodmanClient interface
    - Create network state comparison and change detection
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_ | _Status: New implementation_

  - [x] 4.3 Create VolumeManager following ResourceManager pattern
    - Implement CuteVolume manifest parsing with bind/volume types
    - Add Podman volume operations through PodmanClient interface
    - Create volume state comparison and dependency handling
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_ | _Status: New implementation_

  - [x] 4.4 Create SecretManager following ResourceManager pattern
    - Implement CuteSecret manifest parsing with base64 handling
    - Add Podman secret operations through PodmanClient interface
    - Create secret state comparison and injection logic
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_ | _Status: New implementation_

- [ ] 5. Build centralized ReconciliationController
  - [x] 5.1 Create ReconciliationController interface and implementation
    - Refactor existing chart.Upgrade logic into controller pattern
    - Implement full reconciliation workflow: parse → resolve → compare → execute
    - Add comprehensive error handling and recovery mechanisms
    - _Requirements: 6.1, 6.2, 6.3, 6.4_ | _Status: Refactor existing_

  - [ ] 5.2 Implement orphaned resource cleanup
    - Add detection of resources not present in current manifests
    - Create safe orphan removal respecting dependency order
    - Integrate cleanup reporting with existing output system
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_ | _Status: New implementation_

## Phase 4: Critical Bug Fixes and Core Functionality

- [ ] 6. Fix critical reconciliation and CLI issues (Core functionality fixes)
  - [ ] 6.1 Fix dry-run functionality (Critical Bug Fix)
    - Dry-run mode is not properly preventing actual resource creation/modification
    - Resources are being created even in dry-run mode during state comparison
    - Need to ensure dry-run only performs read operations and planning
    - Fix core dry-run logic to prevent any state modifications
    - _Requirements: 6.5, 10.1_ | _Status: Bug fix_

  - [ ] 6.2 Improve error message readability and actionability
    - Current error messages are too verbose and nested (e.g., "unable to create container: unable to create container: creating container storage...")
    - Error messages should be concise and provide actionable guidance
    - Implement error message truncation and user-friendly formatting
    - Add suggestions for common issues (e.g., "Resource already exists. Run cleanup or use --force flag")
    - _Requirements: 6.3, 6.4_ | _Status: Enhancement_

  - [ ] 6.3 Fix upgrade command functionality (Critical Bug Fix)
    - Upgrade command appears to not be working correctly
    - Should properly detect existing resources and perform updates instead of creates
    - Need to implement proper state comparison for upgrade scenarios
    - Fix core upgrade logic to handle existing resources correctly
    - _Requirements: 10.1, 10.2, 10.3_ | _Status: Bug fix_

  - [ ] 6.4 Implement resource conflict resolution
    - Add detection of existing resources with same names
    - Implement --force flag to override existing resources
    - Add resource cleanup suggestions in error messages
    - Provide clear guidance on resolving naming conflicts
    - _Requirements: 8.1, 8.2, 8.3_ | _Status: Enhancement_
  
## Phase 5: Advanced Features and Enhanced UX

- [ ] 7. Implement comprehensive resource labeling strategy
  - [ ] 7.1 Create consistent labeling system
    - Extend existing cutepod.Namespace label to full strategy
    - Add chart name, version, and managed-by labels to all resources
    - Implement label-based resource querying and filtering
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_ | _Status: Refactor existing_

- [ ] 8. Enhance CLI output formatting system
  - [ ] 8.1 Refactor existing lipgloss output into OutputFormatter interface
    - Create TableFormatter, JSONFormatter, YAMLFormatter implementations
    - Enhance existing tree-style output for all resource types
    - Improve status indicators and progress reporting
    - _Requirements: 6.3, 6.4_ | _Status: Refactor existing_

  - [ ] 8.2 Implement enhanced verbose modes and advanced output features
    - Add verbose mode with timestamps and Podman command logging
    - Create advanced output formatting options (JSON, YAML, table formats)
    - Implement detailed resource inspection and debugging output
    - _Requirements: 6.5_ | _Status: Refactor existing_

- [ ] 9. Integrate ReconciliationController with CLI commands
  - [ ] 9.1 Refactor install command to use ReconciliationController
    - Replace existing chart.Install with ReconciliationController.Reconcile
    - Enhance flag handling and output formatting
    - Add proper namespace handling at CLI level only
    - _Requirements: 6.1, 6.3, 6.5_ | _Status: Refactor existing_

  - [ ] 9.2 Enhance upgrade command with advanced features
    - Add upgrade rollback and history tracking capabilities
    - Implement upgrade validation and pre-flight checks
    - Add upgrade progress tracking and status reporting
    - Enhance upgrade-specific output formatting and user experience
    - _Requirements: 6.1, 6.3, 10.1, 10.2_ | _Status: Enhancement_

## Phase 6: Comprehensive Testing and Error Handling

- [ ] 10. Implement comprehensive error handling and classification
  - [ ] 10.1 Create error classification system
    - Implement ReconciliationError with proper error types
    - Add retry logic for transient Podman API errors
    - Create graceful degradation for partial failures
    - _Requirements: 1.5, 2.5, 4.4, 7.3_ | _Status: Refactor existing_

- [ ] 11. Create comprehensive unit and integration test suite
  - [ ] 11.1 Unit tests for all engines and managers
    - Test StateComparator with various resource configurations
    - Test DependencyResolver with complex dependency scenarios
    - Test all ResourceManagers with mock PodmanClient
    - _Requirements: 10.1, 10.3, 10.5_ | _Status: New implementation_

  - [ ] 11.2 Integration tests for ReconciliationController
    - Test complete reconciliation cycles with real Podman instances
    - Test dependency failure scenarios and recovery
    - Verify idempotency across multiple reconciliation runs
    - _Requirements: 10.1, 10.4, 7.1, 7.2, 7.3_ | _Status: New implementation_

  - [ ] 11.3 CLI and output formatting tests
    - Test all OutputFormatter implementations
    - Test CLI command integration with various scenarios
    - Test error handling and user experience flows
    - _Requirements: 6.3, 6.4, 6.5_ | _Status: New implementation_

  - [ ] 11.4 Bug fix validation tests
    - Test dry-run mode to ensure no actual resource modifications
    - Test error message formatting and readability
    - Test upgrade command with existing resources
    - Test resource conflict resolution scenarios
    - _Requirements: 6.1, 6.2, 6.3, 6.4_ | _Status: New implementation_