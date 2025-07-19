package podman

import (
	"context"
	"fmt"
	"testing"

	"github.com/containers/podman/v5/pkg/specgen"
)

func TestMockPodmanClient_Container(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test connection
	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test container operations
	spec := &ContainerSpec{
		Name:  "test-container",
		Image: "nginx:latest",
		Labels: map[string]string{
			"test": "true",
		},
	}

	// Convert to specgen for mock
	specGen := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image: spec.Image,
		},
	}

	// Create container
	response, err := client.CreateContainer(ctx, specGen)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}

	if response.ID == "" {
		t.Fatal("Container ID should not be empty")
	}

	// Start container
	err = client.StartContainer(ctx, response.ID)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	// List containers
	containers, err := client.ListContainers(ctx, nil, true)
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	if len(containers) != 1 {
		t.Fatalf("Expected 1 container, got %d", len(containers))
	}

	if containers[0].Names[0] != "test-container" {
		t.Fatalf("Expected container name 'test-container', got %s", containers[0].Names[0])
	}

	// Stop container
	err = client.StopContainer(ctx, "test-container", 10)
	if err != nil {
		t.Fatalf("Failed to stop container: %v", err)
	}

	// Remove container
	err = client.RemoveContainer(ctx, "test-container")
	if err != nil {
		t.Fatalf("Failed to remove container: %v", err)
	}

	// Verify container is removed
	containers, err = client.ListContainers(ctx, nil, true)
	if err != nil {
		t.Fatalf("Failed to list containers: %v", err)
	}

	if len(containers) != 0 {
		t.Fatalf("Expected 0 containers after removal, got %d", len(containers))
	}
}

func TestMockPodmanClient_Network(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test network operations
	spec := NetworkSpec{
		Name:   "test-network",
		Driver: "bridge",
		Subnet: "172.20.0.0/16",
		Labels: map[string]string{
			"test": "true",
		},
	}

	// Create network
	network, err := client.CreateNetwork(ctx, spec)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	if network.Name != "test-network" {
		t.Fatalf("Expected network name 'test-network', got %s", network.Name)
	}

	// List networks
	networks, err := client.ListNetworks(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list networks: %v", err)
	}

	if len(networks) != 1 {
		t.Fatalf("Expected 1 network, got %d", len(networks))
	}

	// Inspect network
	inspected, err := client.InspectNetwork(ctx, "test-network")
	if err != nil {
		t.Fatalf("Failed to inspect network: %v", err)
	}

	if inspected.Driver != "bridge" {
		t.Fatalf("Expected driver 'bridge', got %s", inspected.Driver)
	}

	// Remove network
	err = client.RemoveNetwork(ctx, "test-network")
	if err != nil {
		t.Fatalf("Failed to remove network: %v", err)
	}

	// Verify network is removed
	networks, err = client.ListNetworks(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list networks: %v", err)
	}

	if len(networks) != 0 {
		t.Fatalf("Expected 0 networks after removal, got %d", len(networks))
	}
}

func TestMockPodmanClient_Volume(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test volume operations
	spec := VolumeSpec{
		Name:   "test-volume",
		Driver: "local",
		Labels: map[string]string{
			"test": "true",
		},
	}

	// Create volume
	volume, err := client.CreateVolume(ctx, spec)
	if err != nil {
		t.Fatalf("Failed to create volume: %v", err)
	}

	if volume.Name != "test-volume" {
		t.Fatalf("Expected volume name 'test-volume', got %s", volume.Name)
	}

	// List volumes
	volumes, err := client.ListVolumes(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list volumes: %v", err)
	}

	if len(volumes) != 1 {
		t.Fatalf("Expected 1 volume, got %d", len(volumes))
	}

	// Inspect volume
	inspected, err := client.InspectVolume(ctx, "test-volume")
	if err != nil {
		t.Fatalf("Failed to inspect volume: %v", err)
	}

	if inspected.Driver != "local" {
		t.Fatalf("Expected driver 'local', got %s", inspected.Driver)
	}

	// Remove volume
	err = client.RemoveVolume(ctx, "test-volume")
	if err != nil {
		t.Fatalf("Failed to remove volume: %v", err)
	}

	// Verify volume is removed
	volumes, err = client.ListVolumes(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list volumes: %v", err)
	}

	if len(volumes) != 0 {
		t.Fatalf("Expected 0 volumes after removal, got %d", len(volumes))
	}
}

func TestMockPodmanClient_Secret(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test secret operations
	spec := SecretSpec{
		Name: "test-secret",
		Data: []byte("secret-data"),
		Labels: map[string]string{
			"test": "true",
		},
	}

	// Create secret
	secret, err := client.CreateSecret(ctx, spec)
	if err != nil {
		t.Fatalf("Failed to create secret: %v", err)
	}

	if secret.Name != "test-secret" {
		t.Fatalf("Expected secret name 'test-secret', got %s", secret.Name)
	}

	// List secrets
	secrets, err := client.ListSecrets(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(secrets) != 1 {
		t.Fatalf("Expected 1 secret, got %d", len(secrets))
	}

	// Inspect secret
	inspected, err := client.InspectSecret(ctx, "test-secret")
	if err != nil {
		t.Fatalf("Failed to inspect secret: %v", err)
	}

	if inspected.Name != "test-secret" {
		t.Fatalf("Expected secret name 'test-secret', got %s", inspected.Name)
	}

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
	if err != nil {
		t.Fatalf("Failed to update secret: %v", err)
	}

	// Remove secret
	err = client.RemoveSecret(ctx, "test-secret")
	if err != nil {
		t.Fatalf("Failed to remove secret: %v", err)
	}

	// Verify secret is removed
	secrets, err = client.ListSecrets(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(secrets) != 0 {
		t.Fatalf("Expected 0 secrets after removal, got %d", len(secrets))
	}
}

func TestMockPodmanClient_ErrorInjection(t *testing.T) {
	client := NewMockPodmanClient()
	ctx := context.Background()

	// Test connection error
	client.ConnectError = fmt.Errorf("connection failed")
	err := client.Connect(ctx)
	if err == nil {
		t.Fatal("Expected connection error")
	}

	// Reset error and connect
	client.ConnectError = nil
	err = client.Connect(ctx)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Test container creation error
	client.CreateError = fmt.Errorf("create failed")
	_, err = client.CreateContainer(ctx, &specgen.SpecGenerator{})
	if err == nil {
		t.Fatal("Expected create error")
	}

	// Test network creation error
	client.NetworkCreateError = fmt.Errorf("network create failed")
	_, err = client.CreateNetwork(ctx, NetworkSpec{Name: "test"})
	if err == nil {
		t.Fatal("Expected network create error")
	}

	// Test volume creation error
	client.VolumeCreateError = fmt.Errorf("volume create failed")
	_, err = client.CreateVolume(ctx, VolumeSpec{Name: "test"})
	if err == nil {
		t.Fatal("Expected volume create error")
	}

	// Test secret creation error
	client.SecretCreateError = fmt.Errorf("secret create failed")
	_, err = client.CreateSecret(ctx, SecretSpec{Name: "test"})
	if err == nil {
		t.Fatal("Expected secret create error")
	}
}