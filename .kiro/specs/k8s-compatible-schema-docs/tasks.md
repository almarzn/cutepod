# Implementation Plan

- [x] 1. Set up controller-gen integration and basic structure
  - Install controller-gen as a build dependency
  - Create `crds/` directory for generated CRD files
  - Add `make generate` target to Makefile with controller-gen commands
  - _Requirements: 1.1, 4.1_

- [x] 2. Add basic controller-gen markers to resource types
  - [x] 2.1 Add CRD markers to ContainerResource
    - Add `+kubebuilder:object:root=true` and `+kubebuilder:resource` markers
    - Add basic validation markers for required fields (image, ports)
    - Configure API group as `cutepod` and version as `v1alpha1`
    - _Requirements: 1.1, 1.4, 3.1_

  - [x] 2.2 Add CRD markers to NetworkResource
    - Add controller-gen markers for CuteNetwork CRD generation
    - Add basic validation for driver and subnet fields
    - _Requirements: 1.1, 1.4, 3.1_

  - [x] 2.3 Add CRD markers to VolumeResource
    - Add controller-gen markers for CuteVolume CRD generation
    - Add validation for volume type and required fields
    - _Requirements: 1.1, 1.4, 3.1_

  - [x] 2.4 Add CRD markers to SecretResource
    - Add controller-gen markers for CuteSecret CRD generation
    - Add validation for data field structure
    - _Requirements: 1.1, 1.4, 3.1_

  - [x] 2.5 Add CRD markers to PodResource
    - Add controller-gen markers for CutePod CRD generation
    - Add validation for container references
    - _Requirements: 1.1, 1.4, 3.1_

- [ ] 3. Generate CRDs and verify output
- [x] 3.1 Run controller-gen to generate CRDs
  - Execute controller-gen with proper paths and output configuration
  - Verify all 5 CRD files are generated correctly
  - Validate CRD structure follows Kubernetes v1 specification
  - _Requirements: 1.1, 1.2, 3.2_

- [x] 3.2 Test generated CRDs
  - Validate generated CRDs against Kubernetes OpenAPI specification
  - Verify all required fields and validation constraints are present
  - Test that CRDs can be loaded by standard Kubernetes tooling
  - _Requirements: 1.2, 2.2, 6.2_

- [ ] 4. Create simple documentation generator
- [ ] 4.1 Implement basic DocGenerator
  - Create simple Go program to read generated CRDs
  - Extract resource types and basic field information
  - Generate feature-status.md with implementation status
  - _Requirements: 7.1, 7.3_

- [ ] 4.2 Document what's implemented vs what needs testing
  - List all generated CRDs and their validation status
  - Document which features are complete and which need testing
  - Create simple markdown output showing current state
  - _Requirements: 7.3_

- [ ] 5. Basic lint integration (optional for MVP)
- [ ] 5.1 Add CRD validation to lint command
  - Load generated CRDs in lint command
  - Validate basic YAML structure against CRD schemas
  - Provide simple error messages for validation failures
  - _Requirements: 5.1, 5.2_