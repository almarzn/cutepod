package resource

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewVolumePermissionManager(t *testing.T) {
	vpm, err := NewVolumePermissionManager()
	if err != nil {
		t.Fatalf("Failed to create VolumePermissionManager: %v", err)
	}

	if vpm == nil {
		t.Fatal("VolumePermissionManager should not be nil")
	}

	// Basic checks - these will vary by system
	t.Logf("SELinux enabled: %v", vpm.IsSELinuxEnabled())
	t.Logf("Rootless mode: %v", vpm.IsRootlessMode())

	if vpm.IsRootlessMode() && vpm.GetUserNamespaceMapping() == nil {
		t.Error("User namespace mapping should be set in rootless mode")
	}
}

func TestDetectSELinuxEnabled(t *testing.T) {
	vpm := &VolumePermissionManager{}

	// This test will vary by system, so we just ensure it doesn't panic
	enabled := vpm.detectSELinuxEnabled()
	t.Logf("SELinux detected as enabled: %v", enabled)
}

func TestDetectRootlessMode(t *testing.T) {
	vpm := &VolumePermissionManager{}

	rootless := vpm.detectRootlessMode()
	expectedRootless := os.Geteuid() != 0

	if rootless != expectedRootless {
		t.Errorf("Expected rootless mode %v, got %v", expectedRootless, rootless)
	}
}

func TestDetermineSELinuxLabel(t *testing.T) {
	tests := []struct {
		name           string
		seLinuxEnabled bool
		volume         *VolumeResource
		mount          *VolumeMount
		sharedAccess   bool
		expected       string
	}{
		{
			name:           "SELinux disabled",
			seLinuxEnabled: false,
			volume:         createTestVolume("test-vol", nil),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       "",
		},
		{
			name:           "Explicit mount option",
			seLinuxEnabled: true,
			volume:         createTestVolume("test-vol", nil),
			mount:          createTestVolumeMount("test-vol", "/mnt", &VolumeMountOptions{SELinuxLabel: "custom"}),
			sharedAccess:   false,
			expected:       "custom",
		},
		{
			name:           "Volume security context - shared",
			seLinuxEnabled: true,
			volume:         createTestVolume("test-vol", &VolumeSecurityContext{SELinuxOptions: &SELinuxVolumeOptions{Level: "shared"}}),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       "z",
		},
		{
			name:           "Volume security context - private",
			seLinuxEnabled: true,
			volume:         createTestVolume("test-vol", &VolumeSecurityContext{SELinuxOptions: &SELinuxVolumeOptions{Level: "private"}}),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       "Z",
		},
		{
			name:           "Default shared access",
			seLinuxEnabled: true,
			volume:         createTestVolume("test-vol", nil),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   true,
			expected:       "z",
		},
		{
			name:           "Default private access",
			seLinuxEnabled: true,
			volume:         createTestVolume("test-vol", nil),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       "Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := &VolumePermissionManager{
				seLinuxEnabled: tt.seLinuxEnabled,
			}

			result := vpm.DetermineSELinuxLabel(tt.volume, tt.mount, tt.sharedAccess)
			if result != tt.expected {
				t.Errorf("Expected SELinux label %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestHandleUserNamespaceMapping(t *testing.T) {
	tests := []struct {
		name          string
		rootlessMode  bool
		userNSMapping *UserNamespaceMapping
		container     *ContainerResource
		expectError   bool
	}{
		{
			name:          "Rootful mode - no mapping",
			rootlessMode:  false,
			userNSMapping: nil,
			container:     createTestContainer("test-container", int64Ptr(1000), int64Ptr(1000)),
			expectError:   false,
		},
		{
			name:         "Rootless mode with mapping",
			rootlessMode: true,
			userNSMapping: &UserNamespaceMapping{
				UIDMapStart: 100000,
				GIDMapStart: 100000,
				MapSize:     65536,
			},
			container:   createTestContainer("test-container", int64Ptr(1000), int64Ptr(1000)),
			expectError: false,
		},
		{
			name:         "Rootless mode - UID exceeds range",
			rootlessMode: true,
			userNSMapping: &UserNamespaceMapping{
				UIDMapStart: 100000,
				GIDMapStart: 100000,
				MapSize:     1000,
			},
			container:   createTestContainer("test-container", int64Ptr(2000), int64Ptr(500)),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := &VolumePermissionManager{
				rootlessMode:  tt.rootlessMode,
				userNSMapping: tt.userNSMapping,
			}

			volume := createTestVolume("test-vol", nil)
			uidMapping, gidMapping, err := vpm.HandleUserNamespaceMapping(volume, tt.container)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !tt.rootlessMode {
				if uidMapping != nil || gidMapping != nil {
					t.Error("Expected no mappings for rootful mode")
				}
				return
			}

			if tt.userNSMapping != nil {
				if uidMapping == nil || gidMapping == nil {
					t.Error("Expected mappings for rootless mode with user namespace mapping")
					return
				}

				expectedHostUID := tt.userNSMapping.UIDMapStart + *tt.container.Spec.UID
				expectedHostGID := tt.userNSMapping.GIDMapStart + *tt.container.Spec.GID

				if uidMapping.HostID != expectedHostUID {
					t.Errorf("Expected host UID %d, got %d", expectedHostUID, uidMapping.HostID)
				}

				if gidMapping.HostID != expectedHostGID {
					t.Errorf("Expected host GID %d, got %d", expectedHostGID, gidMapping.HostID)
				}
			}
		})
	}
}

func TestManageHostDirectoryOwnership(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "volume-permission-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testPath := filepath.Join(tempDir, "test-dir")
	if err := os.MkdirAll(testPath, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	tests := []struct {
		name          string
		rootlessMode  bool
		userNSMapping *UserNamespaceMapping
		volume        *VolumeResource
		container     *ContainerResource
		expectError   bool
	}{
		{
			name:         "No security context",
			rootlessMode: false,
			volume:       createTestVolume("test-vol", nil),
			container:    createTestContainer("test-container", nil, nil),
			expectError:  false,
		},
		{
			name:         "Security context without owner",
			rootlessMode: false,
			volume:       createTestVolume("test-vol", &VolumeSecurityContext{}),
			container:    createTestContainer("test-container", nil, nil),
			expectError:  false,
		},
		{
			name:         "Rootful mode with ownership",
			rootlessMode: false,
			volume: createTestVolume("test-vol", &VolumeSecurityContext{
				Owner: &VolumeOwnership{
					User:  int64Ptr(int64(os.Geteuid())),
					Group: int64Ptr(int64(os.Getegid())),
				},
			}),
			container:   createTestContainer("test-container", nil, nil),
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := &VolumePermissionManager{
				rootlessMode:  tt.rootlessMode,
				userNSMapping: tt.userNSMapping,
			}

			err := vpm.ManageHostDirectoryOwnership(testPath, tt.volume)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBuildPodmanMountOptions(t *testing.T) {
	tests := []struct {
		name           string
		seLinuxEnabled bool
		volume         *VolumeResource
		mount          *VolumeMount
		sharedAccess   bool
		expected       []string
	}{
		{
			name:           "Basic hostPath volume",
			seLinuxEnabled: false,
			volume:         createTestHostPathVolume("test-vol", "/host/path"),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       []string{"bind"},
		},
		{
			name:           "Read-only mount",
			seLinuxEnabled: false,
			volume:         createTestHostPathVolume("test-vol", "/host/path"),
			mount:          createTestReadOnlyVolumeMount("test-vol", "/mnt"),
			sharedAccess:   false,
			expected:       []string{"bind", "ro"},
		},
		{
			name:           "With SELinux shared",
			seLinuxEnabled: true,
			volume:         createTestHostPathVolume("test-vol", "/host/path"),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   true,
			expected:       []string{"bind", "z"},
		},
		{
			name:           "With SELinux private",
			seLinuxEnabled: true,
			volume:         createTestHostPathVolume("test-vol", "/host/path"),
			mount:          createTestVolumeMount("test-vol", "/mnt", nil),
			sharedAccess:   false,
			expected:       []string{"bind", "Z"},
		},
		{
			name:           "Read-only with SELinux",
			seLinuxEnabled: true,
			volume:         createTestHostPathVolume("test-vol", "/host/path"),
			mount:          createTestReadOnlyVolumeMount("test-vol", "/mnt"),
			sharedAccess:   true,
			expected:       []string{"bind", "ro", "z"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := &VolumePermissionManager{
				seLinuxEnabled: tt.seLinuxEnabled,
			}

			options, err := vpm.BuildPodmanMountOptions(tt.volume, tt.mount, tt.sharedAccess)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if len(options) != len(tt.expected) {
				t.Errorf("Expected %d options, got %d: %v", len(tt.expected), len(options), options)
				return
			}

			for i, expected := range tt.expected {
				if options[i] != expected {
					t.Errorf("Expected option %d to be %q, got %q", i, expected, options[i])
				}
			}
		})
	}
}

func TestDiagnosePermissionError(t *testing.T) {
	tests := []struct {
		name           string
		seLinuxEnabled bool
		rootlessMode   bool
		errorMsg       string
		expectedType   PermissionErrorType
	}{
		{
			name:           "SELinux permission denied",
			seLinuxEnabled: true,
			rootlessMode:   false,
			errorMsg:       "permission denied",
			expectedType:   SELinuxDenied,
		},
		{
			name:           "Operation not permitted",
			seLinuxEnabled: false,
			rootlessMode:   false,
			errorMsg:       "operation not permitted",
			expectedType:   OwnershipMismatch,
		},
		{
			name:           "User namespace error in rootless",
			seLinuxEnabled: false,
			rootlessMode:   true,
			errorMsg:       "user namespace mapping failed",
			expectedType:   UserNSMappingFail,
		},
		{
			name:           "Generic path error",
			seLinuxEnabled: false,
			rootlessMode:   false,
			errorMsg:       "no such file or directory",
			expectedType:   PathNotAccessible,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := &VolumePermissionManager{
				seLinuxEnabled: tt.seLinuxEnabled,
				rootlessMode:   tt.rootlessMode,
			}

			volume := createTestVolume("test-vol", nil)
			mount := createTestVolumeMount("test-vol", "/mnt", nil)

			permErr := vpm.DiagnosePermissionError(
				&testError{msg: tt.errorMsg},
				volume,
				mount,
				"/host/path",
			)

			if permErr.ErrorType != tt.expectedType {
				t.Errorf("Expected error type %s, got %s", tt.expectedType, permErr.ErrorType)
			}

			if permErr.VolumeName != volume.GetName() {
				t.Errorf("Expected volume name %s, got %s", volume.GetName(), permErr.VolumeName)
			}

			if permErr.Suggestion == "" {
				t.Error("Expected non-empty suggestion")
			}
		})
	}
}

// Helper functions for creating test objects

func createTestVolume(name string, securityContext *VolumeSecurityContext) *VolumeResource {
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = name
	volume.Spec.Type = VolumeTypeHostPath
	volume.Spec.HostPath = &HostPathVolumeSource{Path: "/test/path"}
	volume.Spec.SecurityContext = securityContext
	return volume
}

func createTestHostPathVolume(name, path string) *VolumeResource {
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = name
	volume.Spec.Type = VolumeTypeHostPath
	volume.Spec.HostPath = &HostPathVolumeSource{Path: path}
	return volume
}

func createTestVolumeMount(name, mountPath string, options *VolumeMountOptions) *VolumeMount {
	return &VolumeMount{
		Name:         name,
		MountPath:    mountPath,
		ReadOnly:     false,
		MountOptions: options,
	}
}

func createTestReadOnlyVolumeMount(name, mountPath string) *VolumeMount {
	return &VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  true,
	}
}

func createTestContainer(name string, uid, gid *int64) *ContainerResource {
	container := &ContainerResource{}
	container.ObjectMeta.Name = name
	container.Spec.Image = "test:latest"
	container.Spec.UID = uid
	container.Spec.GID = gid
	return container
}

func int64Ptr(i int64) *int64 {
	return &i
}

// testError is a simple error implementation for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
