package resource

import (
	"testing"
)

func TestContainerResource_GetDependencies(t *testing.T) {
	// Test that ContainerResource properly reports its dependencies
	container := NewContainerResource()
	container.ObjectMeta.Name = "web-server"
	container.Spec.Image = "nginx:latest"

	// Add network dependencies
	container.Spec.Networks = []string{"web-network", "db-network"}

	// Add volume dependencies
	container.Spec.Volumes = []VolumeMount{
		{Name: "web-content", MountPath: "/usr/share/nginx/html"},
		{Name: "logs", MountPath: "/var/log/nginx"},
	}

	// Add secret dependencies
	container.Spec.Secrets = []SecretReference{
		{Name: "db-credentials", Env: true},
		{Name: "ssl-certs", Path: "/etc/ssl/certs"},
	}

	// Add pod dependency
	container.Spec.Pod = "web-pod"

	deps := container.GetDependencies()

	// Should have 7 dependencies: 2 networks + 2 volumes + 2 secrets + 1 pod
	if len(deps) != 7 {
		t.Errorf("Expected 7 dependencies, got %d", len(deps))
	}

	// Check that all expected dependencies are present
	expectedDeps := map[string]ResourceType{
		"web-network":    ResourceTypeNetwork,
		"db-network":     ResourceTypeNetwork,
		"web-content":    ResourceTypeVolume,
		"logs":           ResourceTypeVolume,
		"db-credentials": ResourceTypeSecret,
		"ssl-certs":      ResourceTypeSecret,
		"web-pod":        ResourceTypePod,
	}

	actualDeps := make(map[string]ResourceType)
	for _, dep := range deps {
		actualDeps[dep.Name] = dep.Type
	}

	for name, expectedType := range expectedDeps {
		if actualType, exists := actualDeps[name]; !exists {
			t.Errorf("Expected dependency %s not found", name)
		} else if actualType != expectedType {
			t.Errorf("Expected dependency %s to be type %s, got %s", name, expectedType, actualType)
		}
	}
}

func TestContainerResource_GetDependencies_EmptyDependencies(t *testing.T) {
	// Test container with no dependencies
	container := NewContainerResource()
	container.ObjectMeta.Name = "simple-container"
	container.Spec.Image = "nginx:latest"

	deps := container.GetDependencies()

	if len(deps) != 0 {
		t.Errorf("Expected 0 dependencies for container with no dependencies, got %d", len(deps))
	}
}

func TestContainerResource_GetDependencies_PartialDependencies(t *testing.T) {
	// Test container with only some types of dependencies
	container := NewContainerResource()
	container.ObjectMeta.Name = "partial-deps-container"
	container.Spec.Image = "nginx:latest"

	// Only add network and volume dependencies
	container.Spec.Networks = []string{"web-network"}
	container.Spec.Volumes = []VolumeMount{
		{Name: "data-volume", MountPath: "/data"},
	}

	deps := container.GetDependencies()

	// Should have 2 dependencies: 1 network + 1 volume
	if len(deps) != 2 {
		t.Errorf("Expected 2 dependencies, got %d", len(deps))
	}

	// Check that the right dependencies are present
	expectedDeps := map[string]ResourceType{
		"web-network": ResourceTypeNetwork,
		"data-volume": ResourceTypeVolume,
	}

	actualDeps := make(map[string]ResourceType)
	for _, dep := range deps {
		actualDeps[dep.Name] = dep.Type
	}

	for name, expectedType := range expectedDeps {
		if actualType, exists := actualDeps[name]; !exists {
			t.Errorf("Expected dependency %s not found", name)
		} else if actualType != expectedType {
			t.Errorf("Expected dependency %s to be type %s, got %s", name, expectedType, actualType)
		}
	}
}

func TestContainerResource_Validate(t *testing.T) {
	// Test basic validation
	container := NewContainerResource()
	container.Spec.Image = "nginx:latest"

	errors := container.Validate(`
apiVersion: v1
kind: CuteContainer
metadata:
  name: test-container
spec:
  image: nginx:latest
`)

	if len(errors) != 0 {
		t.Errorf("Expected no validation errors for valid container, got %d errors: %v", len(errors), errors)
	}
}

func TestContainerResource_Validate_EmptyImage(t *testing.T) {
	// Test validation with empty image
	container := NewContainerResource()
	container.Spec.Image = ""

	errors := container.Validate(`
apiVersion: v1
kind: CuteContainer
metadata:
  name: test-container
spec:
  image: ""
`)

	if len(errors) == 0 {
		t.Error("Expected validation error for empty image")
	}
}

func TestContainerResource_Validate_InvalidPort(t *testing.T) {
	// Test validation with invalid port
	container := NewContainerResource()
	container.Spec.Image = "nginx:latest"
	container.Spec.Ports = []ContainerPort{
		{ContainerPort: 0}, // Invalid port
	}

	errors := container.Validate(`
apiVersion: v1
kind: CuteContainer
metadata:
  name: test-container
spec:
  image: nginx:latest
  ports:
    - containerPort: 0
`)

	if len(errors) == 0 {
		t.Error("Expected validation error for invalid port")
	}
}
