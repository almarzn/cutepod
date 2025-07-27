# Requirements Document

## Introduction

This feature enhances Cutepod's volume management capabilities by implementing Kubernetes-style volume types and subPath support. The enhancement will provide more flexible and powerful volume mounting options, including hostPath volumes with subPath mounting, emptyDir volumes, and improved bind mount support that separates volume definitions from container specifications.

## Requirements

### Requirement 1

**User Story:** As a developer, I want to define volumes separately from containers using Kubernetes-style volume types, so that I can reuse volume definitions across multiple containers and maintain clean separation of concerns.

#### Acceptance Criteria

1. WHEN a CuteVolume manifest with type "hostPath" is processed THEN the system SHALL create the specified host directory path if it doesn't exist
2. WHEN a CuteVolume manifest with type "emptyDir" is processed THEN the system SHALL create a temporary directory that can be shared between containers
3. WHEN a CuteVolume manifest with type "volume" is processed THEN the system SHALL create a named Podman volume with the specified driver and options
4. WHEN volume specifications include size limits THEN the system SHALL apply appropriate constraints based on the volume type
5. WHEN a hostPath volume is removed THEN the system SHALL NOT delete the host directory (data preservation)
6. WHEN an emptyDir volume is removed THEN the system SHALL delete the temporary directory and all its contents

### Requirement 2

**User Story:** As a developer, I want to mount specific subdirectories of volumes using subPath, so that I can mount only the relevant portions of a volume without exposing the entire volume contents to containers.

#### Acceptance Criteria

1. WHEN a container volume mount specifies a subPath THEN the system SHALL mount only the specified subdirectory from the volume
2. WHEN subPath is used with hostPath volumes THEN the system SHALL resolve the full path by joining the volume's hostPath with the subPath and create the subdirectory if it doesn't exist
3. WHEN subPath is used with emptyDir volumes THEN the system SHALL create the subdirectory within the temporary directory if it doesn't exist
4. WHEN subPath references a path that cannot be created THEN the system SHALL return an error with appropriate permissions information
5. WHEN subPath is empty or not specified THEN the system SHALL mount the entire volume root

### Requirement 3

**User Story:** As a developer, I want to mount individual files from volumes using subPath, so that I can inject specific configuration files without mounting entire directories.

#### Acceptance Criteria

1. WHEN subPath points to a file within a hostPath volume THEN the system SHALL mount that specific file into the container
2. WHEN mounting a file via subPath THEN the system SHALL preserve the file's permissions and ownership where possible
3. WHEN a file specified in subPath doesn't exist THEN the system SHALL return an error during container creation
4. WHEN multiple containers mount the same file via subPath THEN the system SHALL allow shared read access
5. WHEN a file mount is specified as readOnly THEN the system SHALL enforce read-only access within the container

### Requirement 4

**User Story:** As a developer, I want enhanced volume type support including hostPath, emptyDir, and named volumes, so that I can choose the appropriate storage backend for different use cases.

#### Acceptance Criteria

1. WHEN hostPath volume type is used THEN the system SHALL create bind mounts to the specified host directory with proper path validation
2. WHEN emptyDir volume type is used THEN the system SHALL create a temporary directory that is cleaned up when all referencing containers are removed
3. WHEN emptyDir specifies a sizeLimit THEN the system SHALL apply size constraints using appropriate Podman mechanisms
4. WHEN volume type is "volume" THEN the system SHALL create a named Podman volume with persistence across container restarts
5. WHEN volume type is unsupported THEN the system SHALL return a clear error message listing supported types

### Requirement 5

**User Story:** As a developer, I want containers to reference volumes by name with flexible mount options, so that I can mount the same volume at different paths and with different access modes across multiple containers.

#### Acceptance Criteria

1. WHEN a container references a volume by name THEN the system SHALL resolve the volume definition and apply the appropriate mount configuration
2. WHEN the same volume is referenced by multiple containers THEN the system SHALL allow shared access according to each container's mount specifications
3. WHEN a container specifies readOnly for a volume mount THEN the system SHALL enforce read-only access regardless of the volume's default permissions
4. WHEN a volume reference cannot be resolved THEN the system SHALL return an error indicating the missing volume name
5. WHEN volume mount paths conflict within a single container THEN the system SHALL return an error during validation

### Requirement 6

**User Story:** As a developer, I want proper validation of volume specifications and mount configurations, so that I receive clear error messages for invalid configurations before deployment.

#### Acceptance Criteria

1. WHEN hostPath volume specifies a non-existent path THEN the system SHALL validate the path exists during reconciliation
2. WHEN emptyDir volume specifies an invalid sizeLimit format THEN the system SHALL return a validation error with correct format examples
3. WHEN container volume mounts specify invalid mountPath THEN the system SHALL validate the path format and return descriptive errors
4. WHEN subPath contains invalid characters or path traversal attempts THEN the system SHALL reject the configuration with security warnings
5. WHEN volume names contain invalid characters THEN the system SHALL return validation errors following Kubernetes naming conventions

### Requirement 7

**User Story:** As a developer, I want volume dependency resolution to work correctly with the enhanced volume types, so that volumes are created before containers that reference them.

#### Acceptance Criteria

1. WHEN containers reference volumes by name THEN the system SHALL ensure volume resources are processed before container resources
2. WHEN hostPath volumes reference paths that need to be created THEN the system SHALL create the directory structure before container creation
3. WHEN emptyDir volumes are referenced THEN the system SHALL create the temporary directories before starting dependent containers
4. WHEN volume creation fails THEN the system SHALL prevent creation of containers that depend on the failed volume
5. WHEN volumes are deleted THEN the system SHALL stop and remove dependent containers first

### Requirement 8

**User Story:** As a developer, I want proper handling of Podman mount permissions and security contexts, so that volume mounts work correctly with Podman's security model and rootless operation.

#### Acceptance Criteria

1. WHEN mounting hostPath volumes THEN the system SHALL handle Podman's user namespace mapping and provide appropriate UID/GID mapping options
2. WHEN containers run as non-root users THEN the system SHALL ensure volume permissions allow proper access within the container's user context
3. WHEN SELinux is enabled THEN the system SHALL apply appropriate SELinux labels (Z or z flags) to volume mounts based on sharing requirements
4. WHEN rootless Podman is used THEN the system SHALL handle user namespace mapping for volume ownership correctly
5. WHEN permission issues occur THEN the system SHALL provide clear error messages with suggested solutions for common permission problems

### Requirement 9

**User Story:** As a developer, I want comprehensive examples and documentation for the enhanced volume features, so that I can effectively use the new capabilities in my charts.

#### Acceptance Criteria

1. WHEN using hostPath volumes with subPath THEN documentation SHALL provide clear examples showing directory and file mounting scenarios
2. WHEN using emptyDir volumes THEN examples SHALL demonstrate shared temporary storage between containers
3. WHEN combining multiple volume types in a single chart THEN examples SHALL show best practices for organization and naming
4. WHEN troubleshooting volume issues THEN documentation SHALL provide common error scenarios and solutions including permission problems
5. WHEN dealing with Podman-specific mount challenges THEN documentation SHALL provide guidance on SELinux labels, user namespaces, and rootless operation

### Requirement 10

**User Story:** As a developer, I want the enhanced volume system to integrate seamlessly with existing Cutepod features like reconciliation and dry-run mode, so that the new functionality works consistently with the rest of the system.

#### Acceptance Criteria

1. WHEN dry-run mode is used with enhanced volumes THEN the system SHALL show planned volume and mount operations without creating actual resources
2. WHEN reconciliation detects volume configuration changes THEN the system SHALL update mounts appropriately while minimizing container restarts
3. WHEN orphaned volume resources exist THEN the system SHALL clean them up following the same dependency rules as other resources
4. WHEN volume reconciliation fails THEN the system SHALL provide detailed error information and recovery suggestions
5. WHEN enhanced volumes are used with secrets and networks THEN all resource types SHALL work together seamlessly