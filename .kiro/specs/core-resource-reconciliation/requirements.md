# Requirements Document

## Introduction

This feature implements the core resource reconciliation functionality for Cutepod, enabling the system to manage Podman containers, networks, volumes, and secrets through declarative YAML manifests. The reconciliation engine will compare desired state (from charts) with actual state (from Podman) and apply necessary changes to achieve convergence.

## Requirements

### Requirement 1

**User Story:** As a developer, I want Cutepod to automatically create and manage Podman containers based on CuteContainer manifests, so that I can declaratively define my container infrastructure.

#### Acceptance Criteria

1. WHEN a CuteContainer manifest is processed THEN the system SHALL create a corresponding Podman container with the specified image, ports, environment variables, and volume mounts
2. WHEN a CuteContainer specifies resource limits THEN the system SHALL apply CPU and memory constraints to the Podman container
3. WHEN a CuteContainer references secrets THEN the system SHALL inject the secrets as Podman secrets into the container
4. IF a container with the same name already exists THEN the system SHALL compare configurations and update only if changes are detected
5. WHEN container creation fails THEN the system SHALL return a descriptive error message and halt reconciliation

### Requirement 2

**User Story:** As a developer, I want Cutepod to manage Podman networks through CuteNetwork manifests, so that containers can communicate using defined network topologies.

#### Acceptance Criteria

1. WHEN a CuteNetwork manifest is processed THEN the system SHALL create a Podman network with the specified driver and configuration options
2. WHEN network options include subnet configuration THEN the system SHALL apply the subnet settings to the Podman network
3. IF a network with the same name already exists THEN the system SHALL verify the configuration matches and update if necessary
4. WHEN containers reference a network THEN the system SHALL connect the containers to the specified network
5. WHEN network creation fails THEN the system SHALL return an error and prevent dependent container creation

### Requirement 3

**User Story:** As a developer, I want Cutepod to manage persistent storage through CuteVolume manifests, so that container data persists across container restarts.

#### Acceptance Criteria

1. WHEN a CuteVolume manifest is processed THEN the system SHALL create a Podman volume with the specified driver and options
2. WHEN volume type is "bind" THEN the system SHALL create a bind mount to the specified host path
3. WHEN volume type is "volume" THEN the system SHALL create a named Podman volume
4. IF a volume with the same name already exists THEN the system SHALL verify compatibility and reuse the existing volume
5. WHEN containers reference volumes THEN the system SHALL mount the volumes at the specified container paths

### Requirement 4

**User Story:** As a developer, I want Cutepod to manage secrets through CuteSecret manifests, so that sensitive data is securely injected into containers.

#### Acceptance Criteria

1. WHEN a CuteSecret manifest is processed THEN the system SHALL create a Podman secret with base64-decoded data
2. WHEN containers reference secrets THEN the system SHALL make the secrets available as environment variables or mounted files
3. WHEN secret data is updated THEN the system SHALL update the Podman secret and restart dependent containers
4. WHEN secret creation fails THEN the system SHALL return an error and prevent dependent container creation
5. IF a secret with the same name already exists THEN the system SHALL compare content and update only if changed

### Requirement 5

**User Story:** As a developer, I want Cutepod to group containers using CutePod manifests, so that related containers can be managed together with shared restart policies.

#### Acceptance Criteria

1. WHEN a CutePod manifest is processed THEN the system SHALL create all referenced containers as a logical group
2. WHEN restartPolicy is "Always" THEN the system SHALL ensure containers are automatically restarted if they exit
3. WHEN restartPolicy is "OnFailure" THEN the system SHALL restart containers only if they exit with non-zero status
4. WHEN restartPolicy is "Never" THEN the system SHALL not restart containers automatically
5. WHEN any container in a pod fails to start THEN the system SHALL report the error and halt pod creation

### Requirement 6

**User Story:** As a system administrator, I want Cutepod to provide detailed reconciliation status, so that I can understand what changes were applied and troubleshoot issues.

#### Acceptance Criteria

1. WHEN reconciliation begins THEN the system SHALL log the start of the reconciliation process with timestamp
2. WHEN resources are created, updated, or deleted THEN the system SHALL log each action with resource type and name
3. WHEN reconciliation completes successfully THEN the system SHALL report a summary of all changes applied
4. WHEN reconciliation fails THEN the system SHALL provide detailed error messages with context about the failure
5. WHEN dry-run mode is enabled THEN the system SHALL report planned changes without applying them

### Requirement 7

**User Story:** As a developer, I want Cutepod to handle resource dependencies correctly, so that resources are created in the proper order and dependent resources wait for their dependencies.

#### Acceptance Criteria

1. WHEN processing manifests THEN the system SHALL create networks and volumes before containers that reference them
2. WHEN processing manifests THEN the system SHALL create secrets before containers that reference them
3. WHEN a dependency fails to create THEN the system SHALL halt creation of dependent resources
4. WHEN updating resources THEN the system SHALL respect dependency order during updates
5. WHEN deleting resources THEN the system SHALL remove dependent resources before their dependencies

### Requirement 8

**User Story:** As a developer, I want Cutepod to detect and remove orphaned resources, so that resources no longer defined in manifests are cleaned up automatically.

#### Acceptance Criteria

1. WHEN reconciliation runs THEN the system SHALL identify all existing Podman resources tagged with the target namespace label
2. WHEN a labeled resource is not present in the current manifests THEN the system SHALL remove the orphaned resource
3. WHEN removing orphaned containers THEN the system SHALL stop and remove them gracefully
4. WHEN removing orphaned networks THEN the system SHALL disconnect any remaining containers before removal
5. WHEN removing orphaned volumes THEN the system SHALL only remove volumes not referenced by other containers

### Requirement 9

**User Story:** As a developer, I want Cutepod to use consistent labeling for resource tracking, so that resources can be properly identified and managed across reconciliation cycles.

#### Acceptance Criteria

1. WHEN creating any Podman resource THEN the system SHALL apply a namespace label to identify ownership
2. WHEN creating resources THEN the system SHALL apply labels indicating the chart name and version
3. WHEN querying existing resources THEN the system SHALL filter by namespace labels to find managed resources
4. WHEN comparing resources THEN the system SHALL use labels to match desired state with actual state
5. WHEN labels are missing or incorrect THEN the system SHALL update them during reconciliation

### Requirement 10

**User Story:** As a developer, I want Cutepod reconciliation to be idempotent, so that running the same command multiple times produces consistent results without unwanted side effects.

#### Acceptance Criteria

1. WHEN reconciliation is run multiple times with identical manifests THEN the system SHALL detect no changes are needed
2. WHEN no changes are detected THEN the system SHALL not modify existing Podman resources
3. WHEN partial changes exist THEN the system SHALL only update the resources that have changed
4. WHEN reconciliation is interrupted THEN subsequent runs SHALL resume from a consistent state
5. WHEN comparing desired vs actual state THEN the system SHALL ignore non-significant differences like creation timestamps