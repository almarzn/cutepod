package resource

import (
	"strings"
	"testing"
)

func TestVolumeResource_Validate_HostPath(t *testing.T) {
	tests := []struct {
		name        string
		volume      *VolumeResource
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid hostPath volume",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/test",
					},
				},
			},
			expectError: false,
		},
		{
			name: "hostPath volume missing spec",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
				},
			},
			expectError: true,
			errorMsg:    "hostPath specification is required",
		},
		{
			name: "hostPath volume with relative path",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "relative/path",
					},
				},
			},
			expectError: true,
			errorMsg:    "absolute path",
		},
		{
			name: "hostPath volume with path traversal",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/../etc/passwd",
					},
				},
			},
			expectError: true,
			errorMsg:    "cannot contain",
		},
		{
			name: "hostPath volume with invalid type",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/test",
						Type: func() *HostPathType { t := HostPathType("Invalid"); return &t }(),
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid hostPath.type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.volume.Validate()

			if tt.expectError {
				if len(errs) == 0 {
					t.Errorf("Expected validation error, got none")
				} else if tt.errorMsg != "" {
					found := false
					for _, err := range errs {
						if strings.Contains(err.Error(), tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got errors: %v", tt.errorMsg, errs)
					}
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("Expected no validation errors, got: %v", errs)
				}
			}
		})
	}
}

func TestVolumeResource_Validate_EmptyDir(t *testing.T) {
	tests := []struct {
		name        string
		volume      *VolumeResource
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid emptyDir volume",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type:     VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{},
				},
			},
			expectError: false,
		},
		{
			name: "emptyDir volume with valid size limit",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{
						SizeLimit: func() *string { s := "1000000000"; return &s }(),
					},
				},
			},
			expectError: false,
		},
		{
			name: "emptyDir volume missing spec",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeEmptyDir,
				},
			},
			expectError: true,
			errorMsg:    "emptyDir specification is required",
		},
		{
			name: "emptyDir volume with invalid medium",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{
						Medium: "InvalidMedium",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid emptyDir.medium",
		},
		{
			name: "emptyDir volume with invalid size limit",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{
						SizeLimit: func() *string { s := "invalid-size"; return &s }(),
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid emptyDir.sizeLimit format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.volume.Validate()

			if tt.expectError {
				if len(errs) == 0 {
					t.Errorf("Expected validation error, got none")
				} else if tt.errorMsg != "" {
					found := false
					for _, err := range errs {
						if strings.Contains(err.Error(), tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got errors: %v", tt.errorMsg, errs)
					}
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("Expected no validation errors, got: %v", errs)
				}
			}
		})
	}
}

func TestVolumeResource_Validate_SecurityContext(t *testing.T) {
	tests := []struct {
		name        string
		volume      *VolumeResource
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid security context",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/test",
					},
					SecurityContext: &VolumeSecurityContext{
						Owner: &VolumeOwnership{
							User:  func() *int64 { u := int64(1000); return &u }(),
							Group: func() *int64 { g := int64(1000); return &g }(),
						},
						SELinuxOptions: &SELinuxVolumeOptions{
							Level: "shared",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid SELinux level",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/test",
					},
					SecurityContext: &VolumeSecurityContext{
						SELinuxOptions: &SELinuxVolumeOptions{
							Level: "invalid",
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid seLinuxOptions.level",
		},
		{
			name: "negative user ID",
			volume: &VolumeResource{
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: "/tmp/test",
					},
					SecurityContext: &VolumeSecurityContext{
						Owner: &VolumeOwnership{
							User: func() *int64 { u := int64(-1); return &u }(),
						},
					},
				},
			},
			expectError: true,
			errorMsg:    "owner.user must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.volume.Validate()

			if tt.expectError {
				if len(errs) == 0 {
					t.Errorf("Expected validation error, got none")
				} else if tt.errorMsg != "" {
					found := false
					for _, err := range errs {
						if strings.Contains(err.Error(), tt.errorMsg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s', got errors: %v", tt.errorMsg, errs)
					}
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("Expected no validation errors, got: %v", errs)
				}
			}
		})
	}
}

func TestIsValidResourceQuantity(t *testing.T) {
	tests := []struct {
		quantity string
		expected bool
	}{
		{"1Gi", true},
		{"500Mi", true},
		{"1000000000", true},
		{"1.5Gi", true},
		{"100k", true},
		{"", false},
		{"invalid", false},
		{"1.5.5Gi", false},
		{"Gi", false},
		{"1Zi", false}, // Invalid suffix
	}

	for _, tt := range tests {
		t.Run(tt.quantity, func(t *testing.T) {
			result := isValidResourceQuantity(tt.quantity)
			if result != tt.expected {
				t.Errorf("isValidResourceQuantity(%q) = %v, expected %v", tt.quantity, result, tt.expected)
			}
		})
	}
}
