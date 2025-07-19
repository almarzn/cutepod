package podman

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAdapterIntegrationWithExistingCode demonstrates how the abstracted client
// works with existing container operations
func TestAdapterIntegrationWithExistingCode(t *testing.T) {
	// This test demonstrates that existing code can work with either implementation
	// of the PodmanClient interface without modification

	ctx := context.Background()

	// Test function that simulates existing container operations
	testContainerOperations := func(client PodmanClient) error {
		// Connect to Podman
		if err := client.Connect(ctx); err != nil {
			return err
		}
		defer client.Close()

		// Create a container (similar to what Install() does)
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

		response, err := client.CreateContainer(ctx, spec)
		if err != nil {
			return err
		}

		// Start the container
		if err := client.StartContainer(ctx, response.ID); err != nil {
			return err
		}

		// List containers with namespace filter (similar to GetChanges())
		containers, err := client.ListContainers(ctx, map[string][]string{
			"label": {"cutepod.Namespace=test-namespace"},
		}, true)
		if err != nil {
			return err
		}

		// Verify we found our container
		assert.Len(t, containers, 1)
		assert.Equal(t, "test-container", containers[0].Names[0])
		assert.Equal(t, "running", containers[0].State)

		// Inspect the container (similar to ComputeChanges())
		inspect, err := client.InspectContainer(ctx, "test-container")
		if err != nil {
			return err
		}

		assert.Equal(t, "test-container", inspect.Name)
		assert.Equal(t, "nginx:latest", inspect.Image)

		// Stop and remove the container (similar to RemoveContainer())
		if err := client.StopContainer(ctx, "test-container", 10); err != nil {
			return err
		}

		if err := client.RemoveContainer(ctx, "test-container"); err != nil {
			return err
		}

		return nil
	}

	// Test with mock client - this should work perfectly
	t.Run("MockClient", func(t *testing.T) {
		mockClient := NewMockPodmanClient()
		err := testContainerOperations(mockClient)
		assert.NoError(t, err)

		// Verify mock client was called as expected
		assert.Equal(t, 1, mockClient.GetCallCount("Connect"))
		assert.Equal(t, 1, mockClient.GetCallCount("CreateContainer"))
		assert.Equal(t, 1, mockClient.GetCallCount("StartContainer"))
		assert.Equal(t, 1, mockClient.GetCallCount("ListContainers"))
		assert.Equal(t, 1, mockClient.GetCallCount("InspectContainer"))
		assert.Equal(t, 1, mockClient.GetCallCount("StopContainer"))
		assert.Equal(t, 1, mockClient.GetCallCount("RemoveContainer"))
		assert.Equal(t, 1, mockClient.GetCallCount("Close"))
	})

	// Note: We don't test with real PodmanAdapter here because it requires
	// an actual Podman daemon, but the interface ensures compatibility
}

// TestClientProviderUsage demonstrates how to use the client provider pattern
func TestClientProviderUsage(t *testing.T) {
	// This shows how code can be written to work with either real or mock clients

	// Function that accepts a client provider
	performContainerOperation := func(provider ClientProvider) error {
		client := provider.GetClient()

		ctx := context.Background()
		if err := client.Connect(ctx); err != nil {
			return err
		}
		defer client.Close()

		// Pull an image
		if err := client.PullImage(ctx, "nginx:latest"); err != nil {
			return err
		}

		// Check if image exists
		_, err := client.GetImage(ctx, "nginx:latest")
		return err
	}

	// Test with mock provider
	t.Run("MockProvider", func(t *testing.T) {
		mockProvider := NewMockClientProvider()
		err := performContainerOperation(mockProvider)
		assert.NoError(t, err)

		// Verify calls were made
		mockClient := mockProvider.GetMockClient()
		assert.Equal(t, 1, mockClient.GetCallCount("PullImage"))
		assert.Equal(t, 1, mockClient.GetCallCount("GetImage"))
	})

	// Test with default provider (would use real Podman in production)
	t.Run("DefaultProvider", func(t *testing.T) {
		defaultProvider := NewDefaultClientProvider()

		// We can't actually test this without Podman running,
		// but we can verify the provider returns the right type
		client := defaultProvider.GetClient()
		assert.IsType(t, &PodmanAdapter{}, client)
	})
}

// TestErrorHandlingAbstraction tests that error handling works consistently
// across both implementations
func TestErrorHandlingAbstraction(t *testing.T) {
	ctx := context.Background()

	// Test with mock client configured to fail
	mockClient := NewMockPodmanClient()
	mockClient.SetShouldFailConnect(true)

	err := mockClient.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock connection failed")

	// Test operation failure
	mockClient.Reset()
	mockClient.SetShouldFailOperation("CreateContainer", true)

	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
		},
	}

	_, err = mockClient.CreateContainer(ctx, spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock create container failed")
}

// TestResourceOperationsAbstraction tests that all resource types work
// through the abstracted interface
func TestResourceOperationsAbstraction(t *testing.T) {
	ctx := context.Background()
	mockClient := NewMockPodmanClient()

	// Test network operations
	networkSpec := NetworkSpec{
		Name:   "test-network",
		Driver: "bridge",
		Labels: map[string]string{"test": "true"},
	}

	network, err := mockClient.CreateNetwork(ctx, networkSpec)
	require.NoError(t, err)
	assert.Equal(t, "test-network", network.Name)

	networks, err := mockClient.ListNetworks(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, networks, 1)

	// Test volume operations
	volumeSpec := VolumeSpec{
		Name:   "test-volume",
		Driver: "local",
		Labels: map[string]string{"test": "true"},
	}

	volume, err := mockClient.CreateVolume(ctx, volumeSpec)
	require.NoError(t, err)
	assert.Equal(t, "test-volume", volume.Name)

	volumes, err := mockClient.ListVolumes(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, volumes, 1)

	// Test secret operations
	secretSpec := SecretSpec{
		Name:   "test-secret",
		Data:   []byte("secret-data"),
		Labels: map[string]string{"test": "true"},
	}

	secret, err := mockClient.CreateSecret(ctx, secretSpec)
	require.NoError(t, err)
	assert.Equal(t, "test-secret", secret.Name)

	secrets, err := mockClient.ListSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
}
