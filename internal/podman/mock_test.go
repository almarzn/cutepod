package podman

import (
	"context"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockPodmanClient_Connect(t *testing.T) {
	client := NewMockPodmanClient()

	// Test successful connection
	err := client.Connect(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("Connect"))

	// Test failed connection
	client.SetShouldFailConnect(true)
	err = client.Connect(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock connection failed")
}

func TestMockPodmanClient_ContainerOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test container creation
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
	assert.Equal(t, 1, client.GetCallCount("CreateContainer"))

	// Test container start
	err = client.StartContainer(ctx, response.ID)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("StartContainer"))

	// Test container list
	containers, err := client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 1)
	assert.Equal(t, "test-container", containers[0].Names[0])
	assert.Equal(t, "running", containers[0].State)

	// Test container inspect
	inspect, err := client.InspectContainer(ctx, "test-container")
	require.NoError(t, err)
	assert.Equal(t, "test-container", inspect.Name)
	assert.Equal(t, "running", inspect.State.Status)

	// Test container stop
	err = client.StopContainer(ctx, "test-container", 10)
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("StopContainer"))

	// Test container remove
	err = client.RemoveContainer(ctx, "test-container")
	assert.NoError(t, err)
	assert.Equal(t, 1, client.GetCallCount("RemoveContainer"))

	// Verify container is removed
	containers, err = client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 0)
}

func TestMockPodmanClient_NetworkOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test network creation
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

	// Test network list
	networks, err := client.ListNetworks(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, networks, 1)
	assert.Equal(t, "test-network", networks[0].Name)

	// Test network inspect
	inspectNetwork, err := client.InspectNetwork(ctx, "test-network")
	require.NoError(t, err)
	assert.Equal(t, "test-network", inspectNetwork.Name)

	// Test network remove
	err = client.RemoveNetwork(ctx, "test-network")
	assert.NoError(t, err)

	// Verify network is removed
	networks, err = client.ListNetworks(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, networks, 0)
}

func TestMockPodmanClient_VolumeOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test volume creation
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

	// Test volume list
	volumes, err := client.ListVolumes(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, volumes, 1)
	assert.Equal(t, "test-volume", volumes[0].Name)

	// Test volume inspect
	inspectVolume, err := client.InspectVolume(ctx, "test-volume")
	require.NoError(t, err)
	assert.Equal(t, "test-volume", inspectVolume.Name)

	// Test volume remove
	err = client.RemoveVolume(ctx, "test-volume")
	assert.NoError(t, err)

	// Verify volume is removed
	volumes, err = client.ListVolumes(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, volumes, 0)
}

func TestMockPodmanClient_SecretOperations(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test secret creation
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

	// Test secret list
	secrets, err := client.ListSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 1)
	assert.Equal(t, "test-secret", secrets[0].Name)

	// Test secret inspect
	inspectSecret, err := client.InspectSecret(ctx, "test-secret")
	require.NoError(t, err)
	assert.Equal(t, "test-secret", inspectSecret.Name)

	// Test secret update
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

	// Test secret remove
	err = client.RemoveSecret(ctx, "test-secret")
	assert.NoError(t, err)

	// Verify secret is removed
	secrets, err = client.ListSecrets(ctx, nil)
	require.NoError(t, err)
	assert.Len(t, secrets, 0)
}

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

func TestMockPodmanClient_ErrorHandling(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test operation failure
	client.SetShouldFailOperation("CreateContainer", true)

	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name: "test-container",
		},
	}

	_, err := client.CreateContainer(ctx, spec)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mock create container failed")
}

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

	// Reset client
	client.Reset()

	// Verify resources are cleared
	containers, err = client.ListContainers(ctx, nil, true)
	require.NoError(t, err)
	assert.Len(t, containers, 0)

	// Verify call counts are reset
	assert.Equal(t, 0, client.GetCallCount("CreateContainer"))
}
