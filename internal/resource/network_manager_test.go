package resource

import (
	"context"
	"cutepod/internal/labels"
	"cutepod/internal/podman"
	"testing"
)

func TestNetworkManager_ImplementsResourceManager(t *testing.T) {
	// Verify that NetworkManager implements ResourceManager interface
	var _ ResourceManager = &NetworkManager{}
}

func TestNetworkManager_GetResourceType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	if nm.GetResourceType() != ResourceTypeNetwork {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeNetwork, nm.GetResourceType())
	}
}

func TestNetworkManager_GetDesiredState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Create test resources
	network1 := NewNetworkResource()
	network1.ObjectMeta.Name = "test-network-1"
	network1.Spec.Driver = "bridge"
	network1.Spec.Subnet = "172.20.0.0/16"

	network2 := NewNetworkResource()
	network2.ObjectMeta.Name = "test-network-2"
	network2.Spec.Driver = "macvlan"

	container := NewContainerResource()
	container.ObjectMeta.Name = "test-container"

	manifests := []Resource{network1, network2, container}

	desired, err := nm.GetDesiredState(manifests)
	if err != nil {
		t.Fatalf("GetDesiredState failed: %v", err)
	}

	if len(desired) != 2 {
		t.Errorf("Expected 2 network resources, got %d", len(desired))
	}

	// Verify the networks are the right ones
	names := make(map[string]bool)
	for _, res := range desired {
		names[res.GetName()] = true
	}

	if !names["test-network-1"] || !names["test-network-2"] {
		t.Error("Expected networks not found in desired state")
	}
}

func TestNetworkManager_CompareResources(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Create identical networks
	network1 := NewNetworkResource()
	network1.ObjectMeta.Name = "test-network"
	network1.Spec.Driver = "bridge"
	network1.Spec.Subnet = "172.20.0.0/16"
	network1.Spec.Options = map[string]string{"mtu": "1500"}

	network2 := NewNetworkResource()
	network2.ObjectMeta.Name = "test-network"
	network2.Spec.Driver = "bridge"
	network2.Spec.Subnet = "172.20.0.0/16"
	network2.Spec.Options = map[string]string{"mtu": "1500"}

	match, err := nm.CompareResources(network1, network2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if !match {
		t.Error("Expected identical networks to match")
	}

	// Test different drivers
	network2.Spec.Driver = "macvlan"
	match, err = nm.CompareResources(network1, network2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected networks with different drivers to not match")
	}

	// Reset and test different subnets
	network2.Spec.Driver = "bridge"
	network2.Spec.Subnet = "172.21.0.0/16"
	match, err = nm.CompareResources(network1, network2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected networks with different subnets to not match")
	}

	// Reset and test different options
	network2.Spec.Subnet = "172.20.0.0/16"
	network2.Spec.Options = map[string]string{"mtu": "9000"}
	match, err = nm.CompareResources(network1, network2)
	if err != nil {
		t.Fatalf("CompareResources failed: %v", err)
	}

	if match {
		t.Error("Expected networks with different options to not match")
	}
}

func TestNetworkManager_GetActualState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Create a mock network using the proper API
	spec := podman.NetworkSpec{
		Name:   "test-network",
		Driver: "bridge",
		Subnet: "172.20.0.0/16",
		Labels: labels.GetStandardLabels("test-name", "test-version"),
	}

	_, err := mockClient.CreateNetwork(context.Background(), spec)
	if err != nil {
		t.Fatalf("Failed to create mock network: %v", err)
	}

	actual, err := nm.GetActualState(context.Background(), "test-name")
	if err != nil {
		t.Fatalf("GetActualState failed: %v", err)
	}

	if len(actual) != 1 {
		t.Errorf("Expected 1 network, got %d", len(actual))
	}

	if actual[0].GetName() != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", actual[0].GetName())
	}

	networkResource := actual[0].(*NetworkResource)
	if networkResource.Spec.Driver != "bridge" {
		t.Errorf("Expected driver 'bridge', got '%s'", networkResource.Spec.Driver)
	}

	if networkResource.Spec.Subnet != "172.20.0.0/16" {
		t.Errorf("Expected subnet '172.20.0.0/16', got '%s'", networkResource.Spec.Subnet)
	}
}

func TestNetworkManager_CreateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	network := NewNetworkResource()
	network.ObjectMeta.Name = "test-network"
	network.Spec.Driver = "bridge"
	network.Spec.Subnet = "172.20.0.0/16"
	network.Spec.Gateway = "172.20.0.1"
	network.Spec.Options = map[string]string{
		"mtu": "1500",
	}

	err := nm.CreateResource(context.Background(), network)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the network was created
	if mockClient.GetCallCount("CreateNetwork") != 1 {
		t.Errorf("Expected CreateNetwork to be called once, got %d", mockClient.GetCallCount("CreateNetwork"))
	}
}

func TestNetworkManager_UpdateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Create original network
	original := NewNetworkResource()
	original.ObjectMeta.Name = "test-network"
	original.Spec.Driver = "bridge"
	original.Spec.Subnet = "172.20.0.0/16"

	// Create updated network
	updated := NewNetworkResource()
	updated.ObjectMeta.Name = "test-network"
	updated.Spec.Driver = "bridge"
	updated.Spec.Subnet = "172.21.0.0/16"

	// First create the original network
	err := nm.CreateResource(context.Background(), original)
	if err != nil {
		t.Fatalf("Failed to create original network: %v", err)
	}

	// Now update it
	err = nm.UpdateResource(context.Background(), updated, original)
	if err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	// Verify that remove and create were called (update = remove + create)
	if mockClient.GetCallCount("RemoveNetwork") != 1 {
		t.Errorf("Expected RemoveNetwork to be called once, got %d", mockClient.GetCallCount("RemoveNetwork"))
	}

	// Should have 2 CreateNetwork calls: original + updated
	if mockClient.GetCallCount("CreateNetwork") != 2 {
		t.Errorf("Expected CreateNetwork to be called twice, got %d", mockClient.GetCallCount("CreateNetwork"))
	}
}

func TestNetworkManager_DeleteResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	network := NewNetworkResource()
	network.ObjectMeta.Name = "test-network"
	network.Spec.Driver = "bridge"

	// First create the network
	err := nm.CreateResource(context.Background(), network)
	if err != nil {
		t.Fatalf("Failed to create network: %v", err)
	}

	// Now delete it
	err = nm.DeleteResource(context.Background(), network)
	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}

	// Verify that remove was called
	if mockClient.GetCallCount("RemoveNetwork") != 1 {
		t.Errorf("Expected RemoveNetwork to be called once, got %d", mockClient.GetCallCount("RemoveNetwork"))
	}
}

func TestNetworkManager_CompareOptions(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Test identical options
	options1 := map[string]string{"mtu": "1500", "driver": "bridge"}
	options2 := map[string]string{"mtu": "1500", "driver": "bridge"}

	if !nm.compareOptions(options1, options2) {
		t.Error("Expected identical options to match")
	}

	// Test different values
	options2["mtu"] = "9000"
	if nm.compareOptions(options1, options2) {
		t.Error("Expected options with different values to not match")
	}

	// Test different keys
	options2 = map[string]string{"mtu": "1500", "gateway": "192.168.1.1"}
	if nm.compareOptions(options1, options2) {
		t.Error("Expected options with different keys to not match")
	}

	// Test different lengths
	options2 = map[string]string{"mtu": "1500"}
	if nm.compareOptions(options1, options2) {
		t.Error("Expected options with different lengths to not match")
	}

	// Test nil vs empty
	var nilOptions map[string]string
	emptyOptions := make(map[string]string)
	if !nm.compareOptions(nilOptions, emptyOptions) {
		t.Error("Expected nil and empty options to match")
	}
}

func TestNetworkManager_BuildNetworkSpec(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	network := NewNetworkResource()
	network.ObjectMeta.Name = "test-network"
	network.Spec.Driver = "macvlan"
	network.Spec.Subnet = "172.20.0.0/16"
	network.Spec.Options = map[string]string{"mtu": "1500"}
	network.SetLabels(labels.GetStandardLabels("test-name", "test-version"))

	spec := nm.buildNetworkSpec(network)

	if spec.Name != "test-network" {
		t.Errorf("Expected name 'test-network', got '%s'", spec.Name)
	}

	if spec.Driver != "macvlan" {
		t.Errorf("Expected driver 'macvlan', got '%s'", spec.Driver)
	}

	if spec.Subnet != "172.20.0.0/16" {
		t.Errorf("Expected subnet '172.20.0.0/16', got '%s'", spec.Subnet)
	}

	if spec.Options["mtu"] != "1500" {
		t.Errorf("Expected mtu option '1500', got '%s'", spec.Options["mtu"])
	}

	if spec.Labels[labels.LabelChart] != "test-name" {
		t.Errorf("Expected name label 'test-name', got '%s'", spec.Labels[labels.LabelChart])
	}
}

func TestNetworkManager_BuildNetworkSpec_Defaults(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	network := NewNetworkResource()
	network.ObjectMeta.Name = "test-network"
	// No driver specified - should default to bridge

	spec := nm.buildNetworkSpec(network)

	if spec.Driver != "bridge" {
		t.Errorf("Expected default driver 'bridge', got '%s'", spec.Driver)
	}

	if spec.Options == nil {
		t.Error("Expected options map to be initialized")
	}

	if spec.Labels == nil {
		t.Error("Expected labels map to be initialized")
	}
}

func TestNetworkManager_ConvertPodmanNetworkToResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	podmanNetwork := podman.NetworkInfo{
		ID:     "mock-network-id",
		Name:   "test-network",
		Driver: "bridge",
		Subnet: "172.20.0.0/16",
		Options: map[string]string{
			"mtu": "1500",
		},
		Labels: labels.MergeLabels(labels.GetStandardLabels("test-name", "test-version"), map[string]string{"app": "test-app"}),
	}

	resource := nm.convertPodmanNetworkToResource(podmanNetwork)

	if resource.GetName() != "test-network" {
		t.Errorf("Expected name 'test-network', got '%s'", resource.GetName())
	}
	if resource.Spec.Driver != "bridge" {
		t.Errorf("Expected driver 'bridge', got '%s'", resource.Spec.Driver)
	}

	if resource.Spec.Subnet != "172.20.0.0/16" {
		t.Errorf("Expected subnet '172.20.0.0/16', got '%s'", resource.Spec.Subnet)
	}

	if resource.Spec.Options["mtu"] != "1500" {
		t.Errorf("Expected mtu option '1500', got '%s'", resource.Spec.Options["mtu"])
	}

	l := resource.GetLabels()
	if l[labels.LabelChart] != "test-name" {
		t.Errorf("Expected name label 'test-name', got '%s'", l[labels.LabelChart])
	}

	if l["app"] != "test-app" {
		t.Errorf("Expected app label 'test-app', got '%s'", l["app"])
	}
}

func TestNetworkManager_ErrorHandling(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	// Test wrong resource type for CreateResource
	container := NewContainerResource()
	err := nm.CreateResource(context.Background(), container)
	if err == nil {
		t.Error("Expected error when passing wrong resource type to CreateResource")
	}

	// Test wrong resource type for DeleteResource
	err = nm.DeleteResource(context.Background(), container)
	if err == nil {
		t.Error("Expected error when passing wrong resource type to DeleteResource")
	}

	// Test wrong resource type for CompareResources
	network := NewNetworkResource()
	_, err = nm.CompareResources(network, container)
	if err == nil {
		t.Error("Expected error when passing wrong resource type to CompareResources")
	}

	_, err = nm.CompareResources(container, network)
	if err == nil {
		t.Error("Expected error when passing wrong resource type to CompareResources")
	}
}

func TestNetworkManager_PodmanFailures(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	nm := NewNetworkManager(mockClient)

	network := NewNetworkResource()
	network.ObjectMeta.Name = "test-network"

	// Test CreateNetwork failure
	mockClient.SetShouldFailOperation("CreateNetwork", true)
	err := nm.CreateResource(context.Background(), network)
	if err == nil {
		t.Error("Expected error when CreateNetwork fails")
	}
	mockClient.SetShouldFailOperation("CreateNetwork", false)

	// Test ListNetworks failure
	mockClient.SetShouldFailOperation("ListNetworks", true)
	_, err = nm.GetActualState(context.Background(), "test-name")
	if err == nil {
		t.Error("Expected error when ListNetworks fails")
	}
	mockClient.SetShouldFailOperation("ListNetworks", false)

	// Test RemoveNetwork failure
	mockClient.SetShouldFailOperation("RemoveNetwork", true)
	err = nm.DeleteResource(context.Background(), network)
	if err == nil {
		t.Error("Expected error when RemoveNetwork fails")
	}
}
