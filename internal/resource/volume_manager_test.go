package resource

import (
	"context"
	"cutepod/internal/podman"
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

	volume2 := NewVolumeResource()
	volume2.ObjectMeta.Name = "volume2"

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
	volume.Spec.Driver = "local"

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
			Driver: "local",
		},
	}
	volume2 := &VolumeResource{
		BaseResource: BaseResource{ResourceType: ResourceTypeVolume},
		Spec: CuteVolumeSpec{
			Driver: "local",
		},
	}

	match, err := vm.CompareResources(volume1, volume2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical volumes to match")
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
