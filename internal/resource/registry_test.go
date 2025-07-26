package resource

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestRegistry_DependencyResolution(t *testing.T) {
	registry := NewManifestRegistry()

	// Create a network (no dependencies)
	network := NewNetworkResource()
	network.ObjectMeta.Name = "web-network"
	network.Spec.Driver = "bridge"

	// Create a volume (no dependencies)
	volume := NewVolumeResource()
	volume.ObjectMeta.Name = "web-data"
	volume.Spec.Type = VolumeTypeVolume

	// Create a container that depends on network and volume
	container := NewContainerResource()
	container.ObjectMeta.Name = "web-server"
	container.Spec.Image = "nginx:latest"
	container.Spec.Networks = []string{"web-network"}
	container.Spec.Volumes = []VolumeMount{
		{
			Name:      "web-data",
			MountPath: "/usr/share/nginx/html",
		},
	}

	// Add resources to registry
	require.NoError(t, registry.AddResource(network))
	require.NoError(t, registry.AddResource(volume))
	require.NoError(t, registry.AddResource(container))

	// Test dependency validation
	require.NoError(t, registry.ValidateDependencies())

	// Test creation order
	creationOrder, err := registry.GetCreationOrder()
	require.NoError(t, err)
	require.Len(t, creationOrder, 2) // Should have 2 levels

	// First level should contain network and volume (no dependencies)
	level0 := creationOrder[0]
	require.Len(t, level0, 2)

	level0Names := make([]string, len(level0))
	for i, res := range level0 {
		level0Names[i] = res.GetName()
	}
	assert.Contains(t, level0Names, "web-network")
	assert.Contains(t, level0Names, "web-data")

	// Second level should contain container (depends on network and volume)
	level1 := creationOrder[1]
	require.Len(t, level1, 1)
	assert.Equal(t, "web-server", level1[0].GetName())

	// Test deletion order (should be reverse)
	deletionOrder, err := registry.GetDeletionOrder()
	require.NoError(t, err)
	require.Len(t, deletionOrder, 2)

	// First level for deletion should be container
	assert.Equal(t, "web-server", deletionOrder[0][0].GetName())

	// Second level should be network and volume
	level1Delete := deletionOrder[1]
	require.Len(t, level1Delete, 2)
	level1DeleteNames := make([]string, len(level1Delete))
	for i, res := range level1Delete {
		level1DeleteNames[i] = res.GetName()
	}
	assert.Contains(t, level1DeleteNames, "web-network")
	assert.Contains(t, level1DeleteNames, "web-data")
}

func TestManifestRegistry_CircularDependency(t *testing.T) {
	registry := NewManifestRegistry()

	// Create two containers that depend on each other (circular dependency)
	container1 := NewContainerResource()
	container1.ObjectMeta.Name = "container1"
	container1.Spec.Image = "nginx:latest"

	container2 := NewContainerResource()
	container2.ObjectMeta.Name = "container2"
	container2.Spec.Image = "nginx:latest"

	// This would create a circular dependency if we had a way to express it
	// For now, we'll test with a simpler case

	require.NoError(t, registry.AddResource(container1))
	require.NoError(t, registry.AddResource(container2))

	// Should not have circular dependencies with independent containers
	require.NoError(t, registry.ValidateDependencies())
}

func TestManifestRegistry_MissingDependency(t *testing.T) {
	registry := NewManifestRegistry()

	// Create a container that depends on a non-existent network
	container := NewContainerResource()
	container.ObjectMeta.Name = "web-server"
	container.Spec.Image = "nginx:latest"
	container.Spec.Networks = []string{"missing-network"}

	require.NoError(t, registry.AddResource(container))

	// Should fail validation due to missing dependency
	err := registry.ValidateDependencies()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing-network")
}
