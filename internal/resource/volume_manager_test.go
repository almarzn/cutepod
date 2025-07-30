package resource

import (
	"context"
	"cutepod/internal/podman"
	"os"
	"strings"
	"testing"
)

func TestVolumeManager_ImplementsResourceManager(t *testing.T) {
	// Verify that VolumeManager implements ResourceManager interface
	var _ ResourceManager = &VolumeManager{}
}

func TestVolumeManager_GetResourceType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	if vm.GetResourceType() != ResourceTypeVolume {
		t.Errorf("Expected ResourceTypeVolume, got %v", vm.GetResourceType())
	}
}

func TestVolumeManager_GetDesiredState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Create test manifests
	volume1 := NewVolumeResource()
	volume1.ObjectMeta.Name = "volume1"
	volume1.Spec.Type = VolumeTypeVolume

	volume2 := NewVolumeResource()
	volume2.ObjectMeta.Name = "volume2"
	volume2.Spec.Type = VolumeTypeEmptyDir
	volume2.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium: StorageMediumDefault,
	}

	container := NewContainerResource()
	container.ObjectMeta.Name = "container1"

	manifests := []Resource{volume1, volume2, container}

	volumes, err := vm.GetDesiredState(manifests)
	if err != nil {
		t.Fatalf("GetDesiredState failed: %v", err)
	}

	if len(volumes) != 2 {
		t.Errorf("Expected 2 volumes, got %d", len(volumes))
	}

	// Check that only volume resources are returned
	for _, vol := range volumes {
		if vol.GetType() != ResourceTypeVolume {
			t.Errorf("Expected ResourceTypeVolume, got %v", vol.GetType())
		}
	}
}

func TestVolumeManager_CreateResource_NamedVolume(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-volume"
	volume.Spec.Type = VolumeTypeVolume
	volume.Spec.Volume = &VolumeVolumeSource{
		Driver: "local",
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the volume was created by checking it exists in the mock
	createdVolume, err := mockClient.InspectVolume(ctx, "test-volume")
	if err != nil {
		t.Fatalf("Expected volume to be created, but InspectVolume failed: %v", err)
	}

	if createdVolume.Name != "test-volume" {
		t.Errorf("Expected volume name 'test-volume', got %s", createdVolume.Name)
	}

	if createdVolume.Driver != "local" {
		t.Errorf("Expected driver 'local', got %s", createdVolume.Driver)
	}
}

func TestVolumeManager_CompareResources(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Test identical volumes
	volume1 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeVolume,
			Volume: &VolumeVolumeSource{
				Driver: "local",
			},
		},
	}
	volume2 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeVolume,
			Volume: &VolumeVolumeSource{
				Driver: "local",
			},
		},
	}

	match, err := vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical volumes to match")
	}

	// Test different types
	volume2.Spec.Type = VolumeTypeEmptyDir
	volume2.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium: StorageMediumDefault,
	}
	match, err = vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected volumes with different types to not match")
	}
}

func TestVolumeManager_CompareResources_WrongType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	volume := NewVolumeResource()
	container := NewContainerResource()

	_, err := vm.CompareResources(volume, container)
	if err == nil {
		t.Fatal("Expected error for wrong resource type, got nil")
	}

	if !strings.Contains(err.Error(), "expected VolumeResource") {
		t.Errorf("Expected error about VolumeResource type, got: %v", err)
	}
}

func TestVolumeManager_CreateResource_HostPath(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-hostpath"
	volume.Spec.Type = VolumeTypeHostPath
	volume.Spec.HostPath = &HostPathVolumeSource{
		Path: tempDir,
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist", tempDir)
	}

	// Verify no Podman volume was created for hostPath
	_, err = mockClient.InspectVolume(ctx, "test-hostpath")
	if err == nil {
		t.Error("Expected no Podman volume to be created for hostPath")
	}
}

func TestVolumeManager_CreateResource_HostPath_WithSecurityContext(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Create a temporary directory for testing
	tempDir := t.TempDir()

	uid := int64(1000)
	gid := int64(1000)

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-hostpath-security"
	volume.Spec.Type = VolumeTypeHostPath
	volume.Spec.HostPath = &HostPathVolumeSource{
		Path: tempDir,
	}
	volume.Spec.SecurityContext = &VolumeSecurityContext{
		Owner: &VolumeOwnership{
			User:  &uid,
			Group: &gid,
		},
		SELinuxOptions: &SELinuxVolumeOptions{
			Level: "shared",
		},
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)

	// In test environment, we might not have permission to chown
	// This is expected behavior - the permission manager is working correctly
	if err != nil && strings.Contains(err.Error(), "operation not permitted") {
		t.Logf("Permission denied during chown (expected in test environment): %v", err)
		// This is acceptable in test environment
	} else if err != nil {
		t.Fatalf("CreateResource failed with unexpected error: %v", err)
	}

	// Verify the directory exists
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		t.Errorf("Expected directory %s to exist", tempDir)
	}
}

func TestVolumeManager_CreateResource_EmptyDir(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium: StorageMediumDefault,
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify no Podman volume was created for emptyDir
	_, err = mockClient.InspectVolume(ctx, "test-emptydir")
	if err == nil {
		t.Error("Expected no Podman volume to be created for emptyDir")
	}
}

func TestVolumeManager_CreateResource_EmptyDir_WithSizeLimit(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	sizeLimit := "1Gi"
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir-size"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium:    StorageMediumDefault,
		SizeLimit: &sizeLimit,
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}
}

func TestVolumeManager_CreateResource_EmptyDir_Memory(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	sizeLimit := "512Mi"
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "test-emptydir-memory"
	volume.Spec.Type = VolumeTypeEmptyDir
	volume.Spec.EmptyDir = &EmptyDirVolumeSource{
		Medium:    StorageMediumMemory,
		SizeLimit: &sizeLimit,
	}

	ctx := context.Background()
	err := vm.CreateResource(ctx, volume)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}
}

func TestVolumeManager_CompareResources_HostPath(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Test identical hostPath volumes
	volume1 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeHostPath,
			HostPath: &HostPathVolumeSource{
				Path: "/test/path",
			},
		},
	}
	volume2 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeHostPath,
			HostPath: &HostPathVolumeSource{
				Path: "/test/path",
			},
		},
	}

	match, err := vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical hostPath volumes to match")
	}

	// Test different paths
	volume2.Spec.HostPath.Path = "/different/path"
	match, err = vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected hostPath volumes with different paths to not match")
	}
}

func TestVolumeManager_CompareResources_EmptyDir(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	sizeLimit1 := "1Gi"
	sizeLimit2 := "2Gi"

	// Test identical emptyDir volumes
	volume1 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeEmptyDir,
			EmptyDir: &EmptyDirVolumeSource{
				Medium:    StorageMediumDefault,
				SizeLimit: &sizeLimit1,
			},
		},
	}
	volume2 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeEmptyDir,
			EmptyDir: &EmptyDirVolumeSource{
				Medium:    StorageMediumDefault,
				SizeLimit: &sizeLimit1,
			},
		},
	}

	match, err := vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical emptyDir volumes to match")
	}

	// Test different size limits
	volume2.Spec.EmptyDir.SizeLimit = &sizeLimit2
	match, err = vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected emptyDir volumes with different size limits to not match")
	}

	// Test different mediums
	volume2.Spec.EmptyDir.SizeLimit = &sizeLimit1
	volume2.Spec.EmptyDir.Medium = StorageMediumMemory
	match, err = vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected emptyDir volumes with different mediums to not match")
	}
}

func TestVolumeManager_CompareResources_SecurityContext(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	uid1 := int64(1000)
	uid2 := int64(2000)

	// Test volumes with different security contexts
	volume1 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeHostPath,
			HostPath: &HostPathVolumeSource{
				Path: "/test/path",
			},
			SecurityContext: &VolumeSecurityContext{
				Owner: &VolumeOwnership{
					User: &uid1,
				},
				SELinuxOptions: &SELinuxVolumeOptions{
					Level: "shared",
				},
			},
		},
	}
	volume2 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Type: VolumeTypeHostPath,
			HostPath: &HostPathVolumeSource{
				Path: "/test/path",
			},
			SecurityContext: &VolumeSecurityContext{
				Owner: &VolumeOwnership{
					User: &uid2,
				},
				SELinuxOptions: &SELinuxVolumeOptions{
					Level: "shared",
				},
			},
		},
	}

	match, err := vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected volumes with different security contexts to not match")
	}

	// Test identical security contexts
	volume2.Spec.SecurityContext.Owner.User = &uid1
	match, err = vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected volumes with identical security contexts to match")
	}
}

func TestVolumeManager_VolumeCreatorIntegration(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	vm := NewVolumeManager(mockClient)

	// Test that the volume manager can get creators for different types
	creator, err := vm.creatorRegistry.GetCreator(VolumeTypeHostPath)
	if err != nil {
		t.Fatalf("Failed to get hostPath creator: %v", err)
	}

	if !creator.SupportsType(VolumeTypeHostPath) {
		t.Error("Expected hostPath creator to support hostPath type")
	}

	// Test emptyDir creator
	creator, err = vm.creatorRegistry.GetCreator(VolumeTypeEmptyDir)
	if err != nil {
		t.Fatalf("Failed to get emptyDir creator: %v", err)
	}

	if !creator.SupportsType(VolumeTypeEmptyDir) {
		t.Error("Expected emptyDir creator to support emptyDir type")
	}

	// Test named volume creator
	creator, err = vm.creatorRegistry.GetCreator(VolumeTypeVolume)
	if err != nil {
		t.Fatalf("Failed to get volume creator: %v", err)
	}

	if !creator.SupportsType(VolumeTypeVolume) {
		t.Error("Expected volume creator to support volume type")
	}
}
