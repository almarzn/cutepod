package podman

import (
	"context"
	"fmt"
	"sync"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/containers/podman/v5/pkg/specgen"
)

// MockPodmanClient implements PodmanClient for testing
type MockPodmanClient struct {
	mu sync.RWMutex

	// Storage for mock data
	containers map[string]*MockContainer
	networks   map[string]*NetworkInfo
	volumes    map[string]*VolumeInfo
	secrets    map[string]*SecretInfo
	images     map[string]*inspect.ImageData

	// Behavior controls
	shouldFailConnect    bool
	shouldFailOperations map[string]bool

	// Call tracking
	calls map[string]int
}

// MockContainer represents a container in the mock client
type MockContainer struct {
	ID       string
	Name     string
	Image    string
	State    string
	Labels   map[string]string
	Spec     *specgen.SpecGenerator
	Inspect  *define.InspectContainerData
	ListData *types.ListContainer
}

// NewMockPodmanClient creates a new mock Podman client
func NewMockPodmanClient() *MockPodmanClient {
	return &MockPodmanClient{
		containers:           make(map[string]*MockContainer),
		networks:             make(map[string]*NetworkInfo),
		volumes:              make(map[string]*VolumeInfo),
		secrets:              make(map[string]*SecretInfo),
		images:               make(map[string]*inspect.ImageData),
		shouldFailOperations: make(map[string]bool),
		calls:                make(map[string]int),
	}
}

// Connect simulates connecting to Podman
func (m *MockPodmanClient) Connect(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["Connect"]++

	if m.shouldFailConnect {
		return fmt.Errorf("mock connection failed")
	}

	return nil
}

// Close simulates closing the connection
func (m *MockPodmanClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["Close"]++
	return nil
}

// Container operations

// CreateContainer creates a mock container
func (m *MockPodmanClient) CreateContainer(ctx context.Context, spec *specgen.SpecGenerator) (*types.ContainerCreateResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["CreateContainer"]++

	if m.shouldFailOperations["CreateContainer"] {
		return nil, fmt.Errorf("mock create container failed")
	}

	id := fmt.Sprintf("mock-container-%d", len(m.containers))
	name := spec.Name
	if name == "" {
		name = fmt.Sprintf("container-%d", len(m.containers))
	}

	container := &MockContainer{
		ID:     id,
		Name:   name,
		Image:  spec.Image,
		State:  "created",
		Labels: spec.Labels,
		Spec:   spec,
		Inspect: &define.InspectContainerData{
			ID:    id,
			Name:  name,
			Image: spec.Image,
			State: &define.InspectContainerState{
				Status: "created",
			},
			Config: &define.InspectContainerConfig{
				Labels: spec.Labels,
			},
		},
		ListData: &types.ListContainer{
			ID:     id,
			Names:  []string{name},
			Image:  spec.Image,
			State:  "created",
			Labels: spec.Labels,
		},
	}

	m.containers[name] = container

	return &types.ContainerCreateResponse{
		ID: id,
	}, nil
}

// StartContainer starts a mock container
func (m *MockPodmanClient) StartContainer(ctx context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["StartContainer"]++

	if m.shouldFailOperations["StartContainer"] {
		return fmt.Errorf("mock start container failed")
	}

	// Find container by ID or name
	for _, container := range m.containers {
		if container.ID == id || container.Name == id {
			container.State = "running"
			container.Inspect.State.Status = "running"
			container.ListData.State = "running"
			return nil
		}
	}

	return fmt.Errorf("container not found: %s", id)
}

// StopContainer stops a mock container
func (m *MockPodmanClient) StopContainer(ctx context.Context, name string, timeout uint) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["StopContainer"]++

	if m.shouldFailOperations["StopContainer"] {
		return fmt.Errorf("mock stop container failed")
	}

	if container, exists := m.containers[name]; exists {
		container.State = "exited"
		container.Inspect.State.Status = "exited"
		container.ListData.State = "exited"
		return nil
	}

	return fmt.Errorf("container not found: %s", name)
}

// RemoveContainer removes a mock container
func (m *MockPodmanClient) RemoveContainer(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["RemoveContainer"]++

	if m.shouldFailOperations["RemoveContainer"] {
		return fmt.Errorf("mock remove container failed")
	}

	if _, exists := m.containers[name]; exists {
		delete(m.containers, name)
		return nil
	}

	return fmt.Errorf("container not found: %s", name)
}

// ListContainers lists mock containers
func (m *MockPodmanClient) ListContainers(ctx context.Context, filters map[string][]string, all bool) ([]types.ListContainer, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["ListContainers"]++

	if m.shouldFailOperations["ListContainers"] {
		return nil, fmt.Errorf("mock list containers failed")
	}

	var result []types.ListContainer
	for _, container := range m.containers {
		// Apply filters
		if m.matchesFilters(container.Labels, filters) {
			if all || container.State == "running" {
				result = append(result, *container.ListData)
			}
		}
	}

	return result, nil
}

// InspectContainer inspects a mock container
func (m *MockPodmanClient) InspectContainer(ctx context.Context, name string) (*define.InspectContainerData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["InspectContainer"]++

	if m.shouldFailOperations["InspectContainer"] {
		return nil, fmt.Errorf("mock inspect container failed")
	}

	if container, exists := m.containers[name]; exists {
		return container.Inspect, nil
	}

	return nil, fmt.Errorf("container not found: %s", name)
}

// Image operations

// PullImage simulates pulling an image
func (m *MockPodmanClient) PullImage(ctx context.Context, image string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["PullImage"]++

	if m.shouldFailOperations["PullImage"] {
		return fmt.Errorf("mock pull image failed")
	}

	// Add image to mock storage
	m.images[image] = &inspect.ImageData{
		ID: fmt.Sprintf("mock-image-%s", image),
	}

	return nil
}

// GetImage gets mock image information
func (m *MockPodmanClient) GetImage(ctx context.Context, image string) (*inspect.ImageData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["GetImage"]++

	if m.shouldFailOperations["GetImage"] {
		return nil, fmt.Errorf("mock get image failed")
	}

	if imageData, exists := m.images[image]; exists {
		return imageData, nil
	}

	return nil, fmt.Errorf("image not found: %s", image)
}

// Network operations

// CreateNetwork creates a mock network
func (m *MockPodmanClient) CreateNetwork(ctx context.Context, spec NetworkSpec) (*NetworkInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["CreateNetwork"]++

	if m.shouldFailOperations["CreateNetwork"] {
		return nil, fmt.Errorf("mock create network failed")
	}

	network := &NetworkInfo{
		ID:      fmt.Sprintf("mock-network-%s", spec.Name),
		Name:    spec.Name,
		Driver:  spec.Driver,
		Options: spec.Options,
		Subnet:  spec.Subnet,
		Labels:  spec.Labels,
	}

	m.networks[spec.Name] = network
	return network, nil
}

// RemoveNetwork removes a mock network
func (m *MockPodmanClient) RemoveNetwork(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["RemoveNetwork"]++

	if m.shouldFailOperations["RemoveNetwork"] {
		return fmt.Errorf("mock remove network failed")
	}

	if _, exists := m.networks[name]; exists {
		delete(m.networks, name)
		return nil
	}

	return fmt.Errorf("network not found: %s", name)
}

// ListNetworks lists mock networks
func (m *MockPodmanClient) ListNetworks(ctx context.Context, filters map[string][]string) ([]NetworkInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["ListNetworks"]++

	if m.shouldFailOperations["ListNetworks"] {
		return nil, fmt.Errorf("mock list networks failed")
	}

	var result []NetworkInfo
	for _, network := range m.networks {
		if m.matchesFilters(network.Labels, filters) {
			result = append(result, *network)
		}
	}

	return result, nil
}

// InspectNetwork inspects a mock network
func (m *MockPodmanClient) InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["InspectNetwork"]++

	if m.shouldFailOperations["InspectNetwork"] {
		return nil, fmt.Errorf("mock inspect network failed")
	}

	if network, exists := m.networks[name]; exists {
		return network, nil
	}

	return nil, fmt.Errorf("network not found: %s", name)
}

// ConnectContainerToNetwork connects a container to a network (mock)
func (m *MockPodmanClient) ConnectContainerToNetwork(ctx context.Context, containerName, networkName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["ConnectContainerToNetwork"]++

	if m.shouldFailOperations["ConnectContainerToNetwork"] {
		return fmt.Errorf("mock connect container to network failed")
	}

	// Just verify both exist
	if _, exists := m.containers[containerName]; !exists {
		return fmt.Errorf("container not found: %s", containerName)
	}
	if _, exists := m.networks[networkName]; !exists {
		return fmt.Errorf("network not found: %s", networkName)
	}

	return nil
}

// DisconnectContainerFromNetwork disconnects a container from a network (mock)
func (m *MockPodmanClient) DisconnectContainerFromNetwork(ctx context.Context, containerName, networkName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["DisconnectContainerFromNetwork"]++

	if m.shouldFailOperations["DisconnectContainerFromNetwork"] {
		return fmt.Errorf("mock disconnect container from network failed")
	}

	return nil
}

// Volume operations

// CreateVolume creates a mock volume
func (m *MockPodmanClient) CreateVolume(ctx context.Context, spec VolumeSpec) (*VolumeInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["CreateVolume"]++

	if m.shouldFailOperations["CreateVolume"] {
		return nil, fmt.Errorf("mock create volume failed")
	}

	volume := &VolumeInfo{
		Name:       spec.Name,
		Driver:     spec.Driver,
		Mountpoint: fmt.Sprintf("/mock/volumes/%s", spec.Name),
		Options:    spec.Options,
		Labels:     spec.Labels,
	}

	m.volumes[spec.Name] = volume
	return volume, nil
}

// RemoveVolume removes a mock volume
func (m *MockPodmanClient) RemoveVolume(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["RemoveVolume"]++

	if m.shouldFailOperations["RemoveVolume"] {
		return fmt.Errorf("mock remove volume failed")
	}

	if _, exists := m.volumes[name]; exists {
		delete(m.volumes, name)
		return nil
	}

	return fmt.Errorf("volume not found: %s", name)
}

// ListVolumes lists mock volumes
func (m *MockPodmanClient) ListVolumes(ctx context.Context, filters map[string][]string) ([]VolumeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["ListVolumes"]++

	if m.shouldFailOperations["ListVolumes"] {
		return nil, fmt.Errorf("mock list volumes failed")
	}

	var result []VolumeInfo
	for _, volume := range m.volumes {
		if m.matchesFilters(volume.Labels, filters) {
			result = append(result, *volume)
		}
	}

	return result, nil
}

// InspectVolume inspects a mock volume
func (m *MockPodmanClient) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["InspectVolume"]++

	if m.shouldFailOperations["InspectVolume"] {
		return nil, fmt.Errorf("mock inspect volume failed")
	}

	if volume, exists := m.volumes[name]; exists {
		return volume, nil
	}

	return nil, fmt.Errorf("volume not found: %s", name)
}

// Secret operations

// CreateSecret creates a mock secret
func (m *MockPodmanClient) CreateSecret(ctx context.Context, spec SecretSpec) (*SecretInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["CreateSecret"]++

	if m.shouldFailOperations["CreateSecret"] {
		return nil, fmt.Errorf("mock create secret failed")
	}

	secret := &SecretInfo{
		ID:     fmt.Sprintf("mock-secret-%s", spec.Name),
		Name:   spec.Name,
		Labels: spec.Labels,
	}

	m.secrets[spec.Name] = secret
	return secret, nil
}

// UpdateSecret updates a mock secret
func (m *MockPodmanClient) UpdateSecret(ctx context.Context, name string, spec SecretSpec) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["UpdateSecret"]++

	if m.shouldFailOperations["UpdateSecret"] {
		return fmt.Errorf("mock update secret failed")
	}

	if _, exists := m.secrets[name]; exists {
		m.secrets[name] = &SecretInfo{
			ID:     fmt.Sprintf("mock-secret-%s", spec.Name),
			Name:   spec.Name,
			Labels: spec.Labels,
		}
		return nil
	}

	return fmt.Errorf("secret not found: %s", name)
}

// RemoveSecret removes a mock secret
func (m *MockPodmanClient) RemoveSecret(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.calls["RemoveSecret"]++

	if m.shouldFailOperations["RemoveSecret"] {
		return fmt.Errorf("mock remove secret failed")
	}

	if _, exists := m.secrets[name]; exists {
		delete(m.secrets, name)
		return nil
	}

	return fmt.Errorf("secret not found: %s", name)
}

// ListSecrets lists mock secrets
func (m *MockPodmanClient) ListSecrets(ctx context.Context, filters map[string][]string) ([]SecretInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["ListSecrets"]++

	if m.shouldFailOperations["ListSecrets"] {
		return nil, fmt.Errorf("mock list secrets failed")
	}

	var result []SecretInfo
	for _, secret := range m.secrets {
		if m.matchesFilters(secret.Labels, filters) {
			result = append(result, *secret)
		}
	}

	return result, nil
}

// InspectSecret inspects a mock secret
func (m *MockPodmanClient) InspectSecret(ctx context.Context, name string) (*SecretInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.calls["InspectSecret"]++

	if m.shouldFailOperations["InspectSecret"] {
		return nil, fmt.Errorf("mock inspect secret failed")
	}

	if secret, exists := m.secrets[name]; exists {
		return secret, nil
	}

	return nil, fmt.Errorf("secret not found: %s", name)
}

// Test helper methods

// SetShouldFailConnect sets whether Connect should fail
func (m *MockPodmanClient) SetShouldFailConnect(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailConnect = shouldFail
}

// SetShouldFailOperation sets whether a specific operation should fail
func (m *MockPodmanClient) SetShouldFailOperation(operation string, shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFailOperations[operation] = shouldFail
}

// GetCallCount returns the number of times a method was called
func (m *MockPodmanClient) GetCallCount(method string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.calls[method]
}

// Reset clears all mock data and call counts
func (m *MockPodmanClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.containers = make(map[string]*MockContainer)
	m.networks = make(map[string]*NetworkInfo)
	m.volumes = make(map[string]*VolumeInfo)
	m.secrets = make(map[string]*SecretInfo)
	m.images = make(map[string]*inspect.ImageData)
	m.shouldFailOperations = make(map[string]bool)
	m.calls = make(map[string]int)
	m.shouldFailConnect = false
}

// AddMockImage adds a mock image to the client
func (m *MockPodmanClient) AddMockImage(name string, imageData *inspect.ImageData) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.images[name] = imageData
}

// matchesFilters checks if labels match the given filters
func (m *MockPodmanClient) matchesFilters(labels map[string]string, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}

	for filterKey, filterValues := range filters {
		if filterKey == "label" {
			for _, filterValue := range filterValues {
				// Handle label filters in format "key=value"
				for labelKey, labelValue := range labels {
					if filterValue == fmt.Sprintf("%s=%s", labelKey, labelValue) {
						return true
					}
				}
			}
			return false
		}
	}

	return true
}
