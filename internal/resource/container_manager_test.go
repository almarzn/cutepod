package resource

import (
	"context"
	"cutepod/internal/podman"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
)

func TestContainerManager_ImplementsResourceManager(t *testing.T) {
	// Verify that ContainerManager implements ResourceManager interface
	var _ ResourceManager = &ContainerManager{}
}

func TestContainerManager_GetResourceType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	if cm.GetResourceType() != ResourceTypeContainer {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeContainer, cm.GetResourceType())
	}
}

func TestContainerManager_GetDesiredState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	// Create test resources
	container1 := NewContainerResource()
	container1.ObjectMeta.Name = "test-container-1"
	container1.Spec.Image = "nginx:latest"

	container2 := NewContainerResource()
	container2.ObjectMeta.Name = "test-container-2"
	container2.Spec.Image = "redis:latest"

	network := &NetworkResource{
		BaseResource: BaseResource{
			ResourceType: ResourceTypeNetwork,
		},
	}
	network.ObjectMeta.Name = "test-network"

	manifests := []Resource{container1, container2, network}

	desired, err := cm.GetDesiredState(manifests)
	if err != nil {
		t.Fatalf("GetDesiredState failed: %v", err)
	}

	if len(desired) != 2 {
		t.Errorf("Expected 2 container resources, got %d", len(desired))
	}

	// Verify the containers are the right ones
	names := make(map[string]bool)
	for _, res := range desired {
		names[res.GetName()] = true
	}

	if !names["test-container-1"] || !names["test-container-2"] {
		t.Error("Expected containers not found in desired state")
	}
}

func TestContainerManager_CompareResources(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	// Create identical containers
	container1 := NewContainerResource()
	container1.ObjectMeta.Name = "test-container"
	container1.Spec.Image = "nginx:latest"
	container1.Spec.Command = []string{"nginx", "-g", "daemon off;"}
	container1.Spec.Env = []EnvVar{{Name: "ENV1", Value: "value1"}}

	container2 := NewContainerResource()
	container2.ObjectMeta.Name = "test-container"
	container2.Spec.Image = "nginx:latest"
	container2.Spec.Command = []string{"nginx", "-g", "daemon off;"}
	container2.Spec.Env = []EnvVar{{Name: "ENV1", Value: "value1"}}

	match, err := cm.CompareResources(container1, container2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical containers to match")
	}

	// Test different images
	container2.Spec.Image = "nginx:1.20"
	match, err = cm.CompareResources(container1, container2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected containers with different images to not match")
	}
}

func TestContainerManager_GetActualState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	// Create a mock container using the proper API
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
			Labels: map[string]string{
				"cutepod.Namespace": "test-namespace",
			},
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: "nginx:latest",
		},
	}

	_, err := mockClient.CreateContainer(context.Background(), spec)
	if err != nil {
		t.Fatalf("Failed to create mock container: %v", err)
	}

	actual, err := cm.GetActualState(context.Background(), "test-namespace")
	if err != nil {
		t.Fatalf("GetActualState failed: %v", err)
	}

	if len(actual) != 1 {
		t.Errorf("Expected 1 container, got %d", len(actual))
	}

	if actual[0].GetName() != "test-container" {
		t.Errorf("Expected container name 'test-container', got '%s'", actual[0].GetName())
	}
}

func TestContainerManager_CreateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	container := NewContainerResource()
	container.ObjectMeta.Name = "test-container"
	container.ObjectMeta.Namespace = "test-namespace"
	container.Spec.Image = "nginx:latest"
	container.Spec.Ports = []ContainerPort{
		{ContainerPort: 80, HostPort: 8080, Protocol: "TCP"},
	}
	container.Spec.Env = []EnvVar{
		{Name: "ENV1", Value: "value1"},
	}

	err := cm.CreateResource(context.Background(), container)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the container was created
	if mockClient.GetCallCount("CreateContainer") != 1 {
		t.Errorf("Expected CreateContainer to be called once, got %d", mockClient.GetCallCount("CreateContainer"))
	}

	if mockClient.GetCallCount("StartContainer") != 1 {
		t.Errorf("Expected StartContainer to be called once, got %d", mockClient.GetCallCount("StartContainer"))
	}
}

func TestContainerManager_UpdateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	// Create original container
	original := NewContainerResource()
	original.ObjectMeta.Name = "test-container"
	original.ObjectMeta.Namespace = "test-namespace"
	original.Spec.Image = "nginx:1.20"

	// Create updated container
	updated := NewContainerResource()
	updated.ObjectMeta.Name = "test-container"
	updated.ObjectMeta.Namespace = "test-namespace"
	updated.Spec.Image = "nginx:latest"

	// First create the original container
	err := cm.CreateResource(context.Background(), original)
	if err != nil {
		t.Fatalf("Failed to create original container: %v", err)
	}

	// Now update it
	err = cm.UpdateResource(context.Background(), updated, original)
	if err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	// Verify that remove and create were called (update = remove + create)
	if mockClient.GetCallCount("RemoveContainer") != 1 {
		t.Errorf("Expected RemoveContainer to be called once, got %d", mockClient.GetCallCount("RemoveContainer"))
	}

	// Should have 2 CreateContainer calls: original + updated
	if mockClient.GetCallCount("CreateContainer") != 2 {
		t.Errorf("Expected CreateContainer to be called twice, got %d", mockClient.GetCallCount("CreateContainer"))
	}
}

func TestContainerManager_DeleteResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	cm := NewContainerManager(mockClient)

	container := NewContainerResource()
	container.ObjectMeta.Name = "test-container"
	container.ObjectMeta.Namespace = "test-namespace"
	container.Spec.Image = "nginx:latest"

	// First create the container
	err := cm.CreateResource(context.Background(), container)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	// Now delete it
	err = cm.DeleteResource(context.Background(), container)
	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}

	// Verify that stop and remove were called
	if mockClient.GetCallCount("StopContainer") != 1 {
		t.Errorf("Expected StopContainer to be called once, got %d", mockClient.GetCallCount("StopContainer"))
	}

	if mockClient.GetCallCount("RemoveContainer") != 1 {
		t.Errorf("Expected RemoveContainer to be called once, got %d", mockClient.GetCallCount("RemoveContainer"))
	}
}
