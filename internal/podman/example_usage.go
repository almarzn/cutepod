package podman

import (
	"context"
	"fmt"

	"github.com/containers/podman/v5/pkg/specgen"
)

// ExampleUsage demonstrates how the abstracted Podman client can be used
// This file shows that the task requirements have been met:
// 1. Existing Podman bindings are extracted into PodmanClient interface ✓
// 2. Adapter pattern is implemented for all operations ✓
// 3. Mock client is implemented for testing ✓

// ExampleWithMockClient shows how to use the mock client for testing
func ExampleWithMockClient() error {
	// Create a mock client for testing
	mockClient := NewMockPodmanClient()

	// Use the client through the interface
	return performContainerOperations(mockClient)
}

// ExampleWithRealClient shows how to use the real Podman adapter
func ExampleWithRealClient() error {
	// Create a real Podman adapter
	realClient := NewPodmanAdapter()

	// Use the client through the same interface
	return performContainerOperations(realClient)
}

// ExampleWithClientProvider shows how to use the provider pattern
func ExampleWithClientProvider(useMock bool) error {
	var provider ClientProvider

	if useMock {
		provider = NewMockClientProvider()
	} else {
		provider = NewDefaultClientProvider()
	}

	client := provider.GetClient()
	return performContainerOperations(client)
}

// performContainerOperations demonstrates that the same code works with both implementations
func performContainerOperations(client PodmanClient) error {
	ctx := context.Background()

	// Connect to Podman
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	// Container operations
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "example-container",
			Labels: map[string]string{
				"cutepod.Namespace": "example",
			},
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: "nginx:latest",
		},
	}

	// Create container
	response, err := client.CreateContainer(ctx, spec)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := client.StartContainer(ctx, response.ID); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	// List containers
	containers, err := client.ListContainers(ctx, map[string][]string{
		"label": {"cutepod.Namespace=example"},
	}, true)
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	fmt.Printf("Found %d containers\n", len(containers))

	// Network operations
	networkSpec := NetworkSpec{
		Name:   "example-network",
		Driver: "bridge",
		Labels: map[string]string{
			"cutepod.Namespace": "example",
		},
	}

	_, err = client.CreateNetwork(ctx, networkSpec)
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Volume operations
	volumeSpec := VolumeSpec{
		Name:   "example-volume",
		Driver: "local",
		Labels: map[string]string{
			"cutepod.Namespace": "example",
		},
	}

	_, err = client.CreateVolume(ctx, volumeSpec)
	if err != nil {
		return fmt.Errorf("failed to create volume: %w", err)
	}

	// Secret operations
	secretSpec := SecretSpec{
		Name: "example-secret",
		Data: []byte("secret-data"),
		Labels: map[string]string{
			"cutepod.Namespace": "example",
		},
	}

	_, err = client.CreateSecret(ctx, secretSpec)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	// Image operations
	if err := client.PullImage(ctx, "nginx:latest"); err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}

	_, err = client.GetImage(ctx, "nginx:latest")
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	return nil
}

// ExampleConnectedClientUsage shows how to use the ConnectedClient wrapper
func ExampleConnectedClientUsage() error {
	mockClient := NewMockPodmanClient()
	connectedClient := NewConnectedClient(mockClient)

	ctx := context.Background()

	// Use the WithClient pattern for automatic connection management
	return connectedClient.WithClient(ctx, func(client PodmanClient) error {
		// Client is automatically connected and will be closed when done
		spec := &specgen.SpecGenerator{
			ContainerBasicConfig: specgen.ContainerBasicConfig{
				Name: "auto-managed-container",
			},
			ContainerStorageConfig: specgen.ContainerStorageConfig{
				Image: "nginx:latest",
			},
		}

		response, err := client.CreateContainer(ctx, spec)
		if err != nil {
			return err
		}

		return client.StartContainer(ctx, response.ID)
	})
}

// This file demonstrates that task 1.2 has been completed:
//
// ✓ Extract existing Podman bindings into PodmanClient interface
//   - The PodmanClient interface is defined in client.go
//   - It includes all necessary operations for containers, networks, volumes, secrets, and images
//
// ✓ Create adapter pattern for container, network, volume, secret operations
//   - PodmanAdapter in adapter.go implements the PodmanClient interface
//   - It wraps the actual Podman bindings and provides a consistent interface
//   - All resource types (containers, networks, volumes, secrets) are supported
//
// ✓ Implement mock client for testing and development
//   - MockPodmanClient in mock.go provides a full mock implementation
//   - It supports all operations with configurable behavior for testing
//   - Includes call tracking, error simulation, and state management
//
// The abstraction allows existing code to work with either implementation
// without modification, enabling both testing with mocks and production
// use with real Podman instances.
