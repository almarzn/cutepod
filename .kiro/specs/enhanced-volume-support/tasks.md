# Implementation Plan

- [x] 1. Enhance volume type system with Kubernetes-style volume types
  - Update CuteVolumeSpec to support hostPath, emptyDir, and volume types with proper structure
  - Add VolumeSecurityContext, SELinuxVolumeOptions, and VolumeOwnership types
  - Add validation for new volume type specifications
  - _Requirements: 1.1, 1.2, 1.3, 4.1, 4.2, 4.3, 4.4, 4.5_

- [ ] 2. Implement enhanced VolumeMount structure with subPath support
  - Add SubPath field to VolumeMount struct in container.go
  - Add VolumeMountOptions with SELinux and UID/GID mapping support
  - Update container validation to handle new mount options
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 3.1, 3.2, 3.3, 3.4, 3.5_

- [ ] 3. Create volume path management system
  - Implement VolumePathManager for handling directory creation and path resolution
  - Add subPath resolution logic for hostPath and emptyDir volumes
  - Implement file mounting support via subPath
  - Add path validation and security checks to prevent traversal attacks
  - _Requirements: 2.1, 2.2, 2.3, 3.1, 3.2, 6.4_

- [ ] 4. Implement Podman permission management system
  - Create VolumePermissionManager to handle SELinux, user namespaces, and ownership
  - Add SELinux label determination logic (z vs Z flags)
  - Implement user namespace mapping for rootless Podman
  - Add host directory ownership management
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 5. Enhance VolumeManager with new volume type support
  - Update createBindMount to handle hostPath volumes with security context
  - Implement emptyDir volume creation with temporary directory management
  - Add sizeLimit support for emptyDir volumes
  - Update volume comparison logic for new fields
  - _Requirements: 1.1, 1.2, 1.3, 1.5, 1.6, 4.1, 4.2, 4.3_

- [ ] 6. Update ContainerManager to integrate with enhanced volumes
  - Modify convertVolumeMounts to resolve volume references and handle subPath
  - Integrate VolumePermissionManager for proper mount option generation
  - Update buildContainerSpec to use resolved volume paths and security options
  - Add volume dependency validation during container creation
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 7.1, 7.2, 7.3, 7.4, 7.5_

- [ ] 7. Implement comprehensive error handling and diagnostics
  - Create VolumePermissionError type with detailed error classification
  - Add permission error detection and resolution suggestions
  - Implement validation errors for volume specifications and mount configurations
  - Add clear error messages for common permission and configuration issues
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5, 8.5_

- [ ] 8. Create unit tests for volume type system
  - Write tests for new volume type validation and creation
  - Test subPath resolution for different volume types
  - Test permission mapping logic for rootless and rootful modes
  - Test SELinux label determination
  - _Requirements: All requirements - testing coverage_

- [ ] 9. Create integration tests for enhanced volume features
  - Test end-to-end volume mounting with different types and subPath
  - Test permission handling in different Podman configurations
  - Test multi-container volume sharing scenarios
  - Test volume dependency resolution and cleanup
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 10.1, 10.2, 10.3, 10.4, 10.5_

- [ ] 10. Update examples and documentation
  - Create example charts demonstrating hostPath volumes with subPath
  - Add emptyDir volume examples with size limits
  - Document permission handling for different Podman configurations
  - Add troubleshooting guide for common volume issues
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5_