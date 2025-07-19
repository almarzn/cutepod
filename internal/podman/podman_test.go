package podman

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPodmanClientInterface verifies that all implementations satisfy the PodmanClient interface
func TestPodmanClientInterface(t *testing.T) {
	// Test that PodmanAdapter implements PodmanClient interface
	var _ PodmanClient = &PodmanAdapter{}

	// Test that MockPodmanClient implements PodmanClient interface
	var _ PodmanClient = &MockPodmanClient{}
}

// TestClientProvider verifies that ClientProvider interface works
func TestClientProvider(t *testing.T) {
	// Test DefaultClientProvider
	defaultProvider := NewDefaultClientProvider()
	var _ ClientProvider = defaultProvider

	client := defaultProvider.GetClient()
	assert.NotNil(t, client)
	assert.IsType(t, &PodmanAdapter{}, client)

	mockClient := defaultProvider.GetMockClient()
	assert.NotNil(t, mockClient)
	assert.IsType(t, &MockPodmanClient{}, mockClient)

	// Test MockClientProvider
	mockProvider := NewMockClientProvider()
	var _ ClientProvider = mockProvider

	client = mockProvider.GetClient()
	assert.NotNil(t, client)
	assert.IsType(t, &MockPodmanClient{}, client)

	mockClient = mockProvider.GetMockClient()
	assert.NotNil(t, mockClient)
	assert.Same(t, client, mockClient) // Should be the same instance
}

// TestConnectedClient verifies the connected client wrapper
func TestConnectedClient(t *testing.T) {
	mockClient := NewMockPodmanClient()
	connectedClient := NewConnectedClient(mockClient)

	ctx := context.Background()

	// Test getting client (should auto-connect)
	client, err := connectedClient.GetClient(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, client)
	assert.Same(t, mockClient, client)

	// Test WithClient function
	called := false
	err = connectedClient.WithClient(ctx, func(c PodmanClient) error {
		called = true
		assert.Same(t, mockClient, c)

		// Test that we can use the client
		spec := &specgen.SpecGenerator{
			ContainerBasicConfig: specgen.ContainerBasicConfig{
				Name: "test-container",
			},
			ContainerStorageConfig: specgen.ContainerStorageConfig{
				Image: "nginx:latest",
			},
		}

		response, err := c.CreateContainer(ctx, spec)
		require.NoError(t, err)
		assert.NotEmpty(t, response.ID)

		return nil
	})

	assert.NoError(t, err)
	assert.True(t, called)

	// Verify the mock client was called
	assert.Equal(t, 1, mockClient.GetCallCount("Connect"))
	assert.Equal(t, 1, mockClient.GetCallCount("CreateContainer"))

	// Test Close
	err = connectedClient.Close()
	assert.NoError(t, err)
}

// TestMockPodmanClient_BasicOperations tests basic mock client functionality
func TestMockPodmanClient_BasicOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test connection
	err := client.Connect(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("Connect"))

	// Test close
	err = client.Close()
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("Close"))
}

// TestMockPodmanClient_ContainerOperations tests container operations
func TestMockPodmanClient_ContainerOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create container
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
			Labels: map[string]string{
				"app": "test",
			},
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: "nginx:latest",
		},
	}

	response, err := client.CreateContainer(ctx, spec)
	require.NoError(t, err)
	assert.NotEmpty(t, response.ID)
	assert.Equal(t, 1, client.GetCallCount("CreateContainer"))

	// Start container
	err = client.StartContainer(ctx, response.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("StartContainer"))

	// List containers
	containers, err := client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, "test-container", containers[0].Names[0])
	assert.Equal(t, "running", containers[0].State)

	// Inspect container
	inspect, err := client.InspectContainer(ctx, "test-container")
	require.NoError(t, err)
	assert.Equal(t, "test-container", inspect.Name)
	assert.Equal(t, "running", inspect.State.Status)

	// Stop container
	err = client.StopContainer(ctx, "test-container", 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("StopContainer"))

	// Remove container
	err = client.RemoveContainer(ctx, "test-container")
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("RemoveContainer"))

	// Verify container is removed
	containers, err = client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 0)
}

// TestMockPodmanClient_NetworkOperations tests network operations
func TestMockPodmanClient_NetworkOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create network
	spec := NetworkSpec{
		Name:   "test-network",
		Driver: "bridge",
		Subnet: "172.20.0.0/16",
		Labels: map[string]string{
			"test": "true",
		},
	}

	network, err := client.CreateNetwork(ctx, spec)
	require.NoError(t, err)
	assert.Equal(t, "test-network", network.Name)
	assert.Equal(t, "bridge", network.Driver)
	assert.Equal(t, "172.20.0.0/16", network.Subnet)

	// List networks
	networks, err := client.ListNetworks(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, networks, 1)
	assert.Equal(t, "test-network", networks[0].Name)

	// Inspect network
	inspectNetwork, err := client.InspectNetwork(ctx, "test-network")
	require.NoError(t, err)
	assert.Equal(t, "test-network", inspectNetwork.Name)

	// Test container-network operations
	// First create a container
	containerSpec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: "nginx:latest",
		},
	}

	_, err = client.CreateContainer(ctx, containerSpec)
	require.NoError(t, err)

	// Connect container to network
	err = client.ConnectContainerToNetwork(ctx, "test-container", "test-network")
	assert.NoError(t, err)

	// Disconnect container from network
	err = client.DisconnectContainerFromNetwork(ctx, "test-container", "test-network")
	assert.NoError(t, err)

	// Remove network
	err = client.RemoveNetwork(ctx, "test-network")
	assert.NoError(t, err)

	// Verify network is removed
	networks, err = client.ListNetworks(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, networks, 0)
}

// TestMockPodmanClient_VolumeOperations tests volume operations
func TestMockPodmanClient_VolumeOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create volume
	spec := VolumeSpec{
		Name:   "test-volume",
		Driver: "local",
		Labels: map[string]string{
			"test": "true",
		},
	}

	volume, err := client.CreateVolume(ctx, spec)
	require.NoError(t, err)
	assert.Equal(t, "test-volume", volume.Name)
	assert.Equal(t, "local", volume.Driver)
	assert.Contains(t, volume.Mountpoint, "test-volume")

	// List volumes
	volumes, err := client.ListVolumes(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, volumes, 1)
	assert.Equal(t, "test-volume", volumes[0].Name)

	// Inspect volume
	inspectVolume, err := client.InspectVolume(ctx, "test-volume")
	require.NoError(t, err)
	assert.Equal(t, "test-volume", inspectVolume.Name)

	// Remove volume
	err = client.RemoveVolume(ctx, "test-volume")
	assert.NoError(t, err)

	// Verify volume is removed
	volumes, err = client.ListVolumes(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, volumes, 0)
}

// TestMockPodmanClient_SecretOperations tests secret operations
func TestMockPodmanClient_SecretOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create secret
	spec := SecretSpec{
		Name: "test-secret",
		Data: []byte("secret-data"),
		Labels: map[string]string{
			"test": "true",
		},
	}

	secret, err := client.CreateSecret(ctx, spec)
	require.NoError(t, err)
	assert.Equal(t, "test-secret", secret.Name)
	assert.NotEmpty(t, secret.ID)

	// List secrets
	secrets, err := client.ListSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
	assert.Equal(t, "test-secret", secrets[0].Name)

	// Inspect secret
	inspectSecret, err := client.InspectSecret(ctx, "test-secret")
	require.NoError(t, err)
	assert.Equal(t, "test-secret", inspectSecret.Name)

	// Update secret
	updateSpec := SecretSpec{
		Name: "test-secret",
		Data: []byte("updated-secret-data"),
		Labels: map[string]string{
			"test":    "true",
			"updated": "true",
		},
	}

	err = client.UpdateSecret(ctx, "test-secret", updateSpec)
	assert.NoError(t, err)

	// Remove secret
	err = client.RemoveSecret(ctx, "test-secret")
	assert.NoError(t, err)

	// Verify secret is removed
	secrets, err = client.ListSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)
}

// TestMockPodmanClient_ImageOperations tests image operations
func TestMockPodmanClient_ImageOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test image pull
	err := client.PullImage(ctx, "nginx:latest")
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("PullImage"))

	// Test image get
	image, err := client.GetImage(ctx, "nginx:latest")
	require.NoError(t, err)
	assert.NotEmpty(t, image.ID)
	assert.Contains(t, image.ID, "nginx:latest")
}

// TestMockPodmanClient_ErrorHandling tests error injection and handling
func TestMockPodmanClient_ErrorHandling(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test connection failure
	client.SetShouldFailConnect(true)
	err := client.Connect(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock connection failed")

	// Reset and test operation failure
	client.Reset()
	client.SetShouldFailOperation("CreateContainer", true)

	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
		},
	}

	_, err = client.CreateContainer(ctx, spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock create container failed")

	// Test other operation failures
	client.Reset()
	client.SetShouldFailOperation("CreateNetwork", true)
	_, err = client.CreateNetwork(ctx, NetworkSpec{Name: "test"})
	assert.Error(t, err)

	client.Reset()
	client.SetShouldFailOperation("CreateVolume", true)
	_, err = client.CreateVolume(ctx, VolumeSpec{Name: "test"})
	assert.Error(t, err)

	client.Reset()
	client.SetShouldFailOperation("CreateSecret", true)
	_, err = client.CreateSecret(ctx, SecretSpec{Name: "test"})
	assert.Error(t, err)
}

// TestMockPodmanClient_FilterMatching tests label filtering
func TestMockPodmanClient_FilterMatching(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create container with labels
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
			Labels: map[string]string{
				"app":       "test",
				"namespace": "default",
			},
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: "nginx:latest",
		},
	}

	_, err := client.CreateContainer(ctx, spec)
	require.NoError(t, err)

	// Test filtering by label
	filters := map[string][]string{
		"label": {"namespace=default"},
	}

	containers, err := client.ListContainers(ctx, filters, true)
	require.NoError(t, err)
	assert.Len(t, containers, 1)

	// Test filtering with non-matching label
	filters = map[string][]string{
		"label": {"namespace=other"},
	}

	containers, err = client.ListContainers(ctx, filters, true)
	require.NoError(t, err)
	assert.Len(t, containers, 0)
}

// TestMockPodmanClient_Reset tests the reset functionality
func TestMockPodmanClient_Reset(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Create some resources
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
		},
	}

	_, err := client.CreateContainer(ctx, spec)
	require.NoError(t, err)

	// Verify resources exist
	containers, err := client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 1)

	// Verify call count
	assert.Equal(t, 1, client.GetCallCount("CreateContainer"))

	// Reset client
	client.Reset()

	// Verify resources are cleared
	containers, err = client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 0)

	// Verify call counts are reset
	assert.Equal(t, 0, client.GetCallCount("CreateContainer"))
}

// TestResourceSpecs verifies that resource specification types are properly defined
func TestResourceSpecs(t *testing.T) {
	// Test ContainerSpec
	containerSpec := ContainerSpec{
		Name:  "test-container",
		Image: "nginx:latest",
		Env:   map[string]string{"ENV": "test"},
		Ports: []PortMapping{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		},
		Volumes: []VolumeMount{
			{Source: "/host", Destination: "/container", ReadOnly: false},
		},
		Labels: map[string]string{"app": "test"},
	}

	assert.Equal(t, "test-container", containerSpec.Name)
	assert.Equal(t, "nginx:latest", containerSpec.Image)

	// Test NetworkSpec
	networkSpec := NetworkSpec{
		Name:    "test-network",
		Driver:  "bridge",
		Options: map[string]string{"subnet": "172.20.0.0/16"},
		Subnet:  "172.20.0.0/16",
		Labels:  map[string]string{"test": "true"},
	}

	assert.Equal(t, "test-network", networkSpec.Name)
	assert.Equal(t, "bridge", networkSpec.Driver)

	// Test VolumeSpec
	volumeSpec := VolumeSpec{
		Name:     "test-volume",
		Driver:   "local",
		Options:  map[string]string{"type": "tmpfs"},
		Labels:   map[string]string{"test": "true"},
		HostPath: "/host/path",
	}

	assert.Equal(t, "test-volume", volumeSpec.Name)
	assert.Equal(t, "local", volumeSpec.Driver)

	// Test SecretSpec
	secretSpec := SecretSpec{
		Name:   "test-secret",
		Data:   []byte("secret-data"),
		Labels: map[string]string{"test": "true"},
	}

	assert.Equal(t, "test-secret", secretSpec.Name)
	assert.Equal(t, "secret-data", string(secretSpec.Data))
}

// TestResourceInfo verifies that resource info types are properly defined
func TestResourceInfo(t *testing.T) {
	// Test NetworkInfo
	networkInfo := NetworkInfo{
		ID:      "network-123",
		Name:    "test-network",
		Driver:  "bridge",
		Options: map[string]string{"subnet": "172.20.0.0/16"},
		Subnet:  "172.20.0.0/16",
		Labels:  map[string]string{"test": "true"},
	}

	assert.Equal(t, "test-network", networkInfo.Name)
	assert.Equal(t, "network-123", networkInfo.ID)

	// Test VolumeInfo
	volumeInfo := VolumeInfo{
		Name:       "test-volume",
		Driver:     "local",
		Mountpoint: "/var/lib/containers/storage/volumes/test-volume",
		Options:    map[string]string{"type": "tmpfs"},
		Labels:     map[string]string{"test": "true"},
	}

	assert.Equal(t, "test-volume", volumeInfo.Name)
	assert.Equal(t, "local", volumeInfo.Driver)

	// Test SecretInfo
	secretInfo := SecretInfo{
		ID:     "secret-123",
		Name:   "test-secret",
		Labels: map[string]string{"test": "true"},
	}

	assert.Equal(t, "test-secret", secretInfo.Name)
	assert.Equal(t, "secret-123", secretInfo.ID)
}
