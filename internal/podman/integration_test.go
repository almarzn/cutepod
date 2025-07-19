package podman

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClientProviderIntegration tests that the client provider pattern works correctly
func TestClientProviderIntegration(t *testing.T) {
	// Test DefaultClientProvider
	defaultProvider := NewDefaultClientProvider()

	// Get real client (this will be a PodmanAdapter)
	realClient := defaultProvider.GetClient()
	assert.NotNil(t, realClient)

	// Get mock client
	mockClient := defaultProvider.GetMockClient()
	assert.NotNil(t, mockClient)
	assert.IsType(t, &MockPodmanClient{}, mockClient)

	// Test MockClientProvider
	mockProvider := NewMockClientProvider()

	// Get client (this will be a MockPodmanClient)
	client := mockProvider.GetClient()
	assert.NotNil(t, client)

	// Get mock client (same instance)
	mockClient2 := mockProvider.GetMockClient()
	assert.NotNil(t, mockClient2)
	assert.Same(t, client, mockClient2) // Should be the same instance
}

// TestConnectedClientWrapper tests the ConnectedClient wrapper
func TestConnectedClientWrapper(t *testing.T) {
	mockClient := NewMockPodmanClient()
	connectedClient := NewConnectedClient(mockClient)

	ctx := context.Background()

	// Test WithClient pattern
	err := connectedClient.WithClient(ctx, func(client PodmanClient) error {
		// Verify we can use the client
		spec := &specgen.SpecGenerator{
			ContainerBasicConfig: specgen.ContainerBasicConfig{
				Name: "test-container",
			},
			ContainerStorageConfig: specgen.ContainerStorageConfig{
				Image: "nginx:latest",
			},
		}

		response, err := client.CreateContainer(ctx, spec)
		require.NoError(t, err)
		assert.NotEmpty(t, response.ID)

		return nil
	})

	assert.NoError(t, err)

	// Verify the mock client was called
	assert.Equal(t, 1, mockClient.GetCallCount("Connect"))
	assert.Equal(t, 1, mockClient.GetCallCount("CreateContainer"))
}

// TestPodmanClientInterface verifies that both implementations satisfy the interface
func TestPodmanClientInterface(t *testing.T) {
	// Test that MockPodmanClient implements PodmanClient
	var client PodmanClient = NewMockPodmanClient()
	assert.NotNil(t, client)

	// Test that PodmanAdapter implements PodmanClient
	var adapter PodmanClient = NewPodmanAdapter()
	assert.NotNil(t, adapter)
}

// TestClientAbstraction tests that we can use either client implementation interchangeably
func TestClientAbstraction(t *testing.T) {
	ctx := context.Background()

	// Test with mock client
	testWithClient := func(client PodmanClient) error {
		// Connect
		err := client.Connect(ctx)
		if err != nil {
			return err
		}
		defer client.Close()

		// Create container
		spec := &specgen.SpecGenerator{
			ContainerBasicConfig: specgen.ContainerBasicConfig{
				Name: "test-container",
			},
			ContainerStorageConfig: specgen.ContainerStorageConfig{
				Image: "nginx:latest",
			},
		}

		response, err := client.CreateContainer(ctx, spec)
		if err != nil {
			return err
		}

		// Start container
		err = client.StartContainer(ctx, response.ID)
		if err != nil {
			return err
		}

		// List containers
		containers, err := client.ListContainers(ctx, nil, true)
		if err != nil {
			return err
		}

		assert.Len(t, containers, 1)
		assert.Equal(t, "test-container", containers[0].Names[0])

		return nil
	}

	// Test with mock client
	mockClient := NewMockPodmanClient()
	err := testWithClient(mockClient)
	assert.NoError(t, err)

	// Note: We can't test with real PodmanAdapter here because it requires
	// an actual Podman daemon running, but the interface abstraction works
}

// TestAdapterPattern tests that the adapter pattern is properly implemented
func TestAdapterPattern(t *testing.T) {
	// The adapter should provide a consistent interface regardless of the underlying implementation

	// Mock implementation
	mockClient := NewMockPodmanClient()

	// Real implementation (adapter)
	realClient := NewPodmanAdapter()

	// Both should implement the same interface
	var client1 PodmanClient = mockClient
	var client2 PodmanClient = realClient

	assert.NotNil(t, client1)
	assert.NotNil(t, client2)

	// Both should have the same method signatures
	ctx := context.Background()

	// Test that methods exist and have correct signatures (compile-time check)
	_ = client1.Connect(ctx)
	_ = client1.Close()

	_ = client2.Connect(ctx)
	_ = client2.Close()
}
