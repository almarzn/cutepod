package resource

import (
	"context"
	"cutepod/internal/podman"
	"path/filepath"
	"testing"
)

func TestVolumeCreatorRegistry_GetCreator(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()
	registry := NewVolumeCreatorRegistry(pathManager, permissionMgr)

	tests := []struct {
		name       string
		volumeType VolumeType
		wantError  bool
	}{
		{
			name:       "hostPath volume",
			volumeType: VolumeTypeHostPath,
			wantError:  false,
		},
		{
			name:       "emptyDir volume",
			volumeType: VolumeTypeEmptyDir,
			wantError:  false,
		},
		{
			name:       "named volume",
			volumeType: VolumeTypeVolume,
			wantError:  false,
		},
		{
			name:       "unsupported volume type",
			volumeType: VolumeType("unsupported"),
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creator, err := registry.GetCreator(tt.volumeType)
			if tt.wantError {
				if err == nil {
					t.Error("Expected error for unsupported volume type")
				}
				return
			}

			if err != nil {
				t.Fatalf("GetCreator failed: %v", err)
			}

			if !creator.SupportsType(tt.volumeType) {
				t.Errorf("Creator should support volume type %s", tt.volumeType)
			}
		})
	}
}

func TestHostPathVolumeCreator_CreateVolume(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()
	creator := NewHostPathVolumeCreator(pathManager, permissionMgr)

	tempDir := t.TempDir()

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-hostpath"
	volume.Spec.Type = VolumeTypeHostPath
	volume.Spec.HostPath = &HostPathVolumeSource{
		Path: tempDir,
	}

	ctx := context.Background()
	mockClient := podman.NewMockPodmanClient()
	pathInfo, err := creator.CreateVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if pathInfo.SourcePath != tempDir {
		t.Errorf("Expected source path %s, got %s", tempDir, pathInfo.SourcePath)
	}

	if pathInfo.RequiresCreation {
		t.Error("Expected existing directory to not require creation")
	}

	if pathInfo.IsFile {
		t.Error("Expected directory, not file")
	}
}

func TestEmptyDirVolumeCreator_CreateVolume(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()
	creator := NewEmptyDirVolumeCreator(pathManager, permissionMgr)

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium: StorageMediumDefault,
	}

	ctx := context.Background()
	mockClient := podman.NewMockPodmanClient()
	pathInfo, err := creator.CreateVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	expectedPath := filepath.Join(pathManager.tempDirBase, "test-emptydir")
	if pathInfo.SourcePath != expectedPath {
		t.Errorf("Expected source path %s, got %s", expectedPath, pathInfo.SourcePath)
	}

	if !pathInfo.RequiresCreation {
		t.Error("Expected emptyDir volume to require creation")
	}

	if pathInfo.IsFile {
		t.Error("Expected directory, not file")
	}

	// Test deletion
	err = creator.DeleteVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
}

func TestEmptyDirVolumeCreator_CreateVolume_WithSizeLimit(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()
	creator := NewEmptyDirVolumeCreator(pathManager, permissionMgr)

	sizeLimit := "1Gi"
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir-size"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium:    StorageMediumDefault,
		SizeLimit: &sizeLimit,
	}

	ctx := context.Background()
	mockClient := podman.NewMockPodmanClient()
	pathInfo, err := creator.CreateVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if pathInfo == nil {
		t.Fatal("Expected pathInfo to be returned")
	}
}

func TestEmptyDirVolumeCreator_CreateVolume_Memory(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()
	creator := NewEmptyDirVolumeCreator(pathManager, permissionMgr)

	sizeLimit := "512Mi"
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir-memory"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium:    StorageMediumMemory,
		SizeLimit: &sizeLimit,
	}

	ctx := context.Background()
	mockClient := podman.NewMockPodmanClient()
	pathInfo, err := creator.CreateVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if pathInfo == nil {
		t.Fatal("Expected pathInfo to be returned")
	}
}

func TestNamedVolumeCreator_CreateVolume(t *testing.T) {
	creator := NewNamedVolumeCreator()

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-named-volume"
	volume.Spec.Type = VolumeTypeVolume
	volume.Spec.Volume = &VolumeVolumeSource{
		Driver: "local",
	}

	ctx := context.Background()
	mockClient := podman.NewMockPodmanClient()
	pathInfo, err := creator.CreateVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("CreateVolume failed: %v", err)
	}

	if pathInfo.SourcePath != "test-named-volume" {
		t.Errorf("Expected source path 'test-named-volume', got %s", pathInfo.SourcePath)
	}

	if pathInfo.RequiresCreation {
		t.Error("Expected named volume to not require creation")
	}

	if pathInfo.IsFile {
		t.Error("Expected directory, not file")
	}

	// Test deletion
	err = creator.DeleteVolume(ctx, mockClient, volume)
	if err != nil {
		t.Fatalf("DeleteVolume failed: %v", err)
	}
}

func TestVolumeCreator_SupportsType(t *testing.T) {
	pathManager := NewVolumePathManager("")
	permissionMgr, _ := NewVolumePermissionManager()

	tests := []struct {
		name       string
		creator    VolumeCreator
		volumeType VolumeType
		expected   bool
	}{
		{
			name:       "hostPath creator supports hostPath",
			creator:    NewHostPathVolumeCreator(pathManager, permissionMgr),
			volumeType: VolumeTypeHostPath,
			expected:   true,
		},
		{
			name:       "hostPath creator doesn't support emptyDir",
			creator:    NewHostPathVolumeCreator(pathManager, permissionMgr),
			volumeType: VolumeTypeEmptyDir,
			expected:   false,
		},
		{
			name:       "emptyDir creator supports emptyDir",
			creator:    NewEmptyDirVolumeCreator(pathManager, permissionMgr),
			volumeType: VolumeTypeEmptyDir,
			expected:   true,
		},
		{
			name:       "emptyDir creator doesn't support volume",
			creator:    NewEmptyDirVolumeCreator(pathManager, permissionMgr),
			volumeType: VolumeTypeVolume,
			expected:   false,
		},
		{
			name:       "named volume creator supports volume",
			creator:    NewNamedVolumeCreator(),
			volumeType: VolumeTypeVolume,
			expected:   true,
		},
		{
			name:       "named volume creator doesn't support hostPath",
			creator:    NewNamedVolumeCreator(),
			volumeType: VolumeTypeHostPath,
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.creator.SupportsType(tt.volumeType)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
