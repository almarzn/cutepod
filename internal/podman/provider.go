package podman

import (
	"context"
	"fmt"
)

// ClientProvider provides Podman clients
type ClientProvider interface {
	GetClient() PodmanClient
	GetMockClient() *MockPodmanClient
}

// DefaultClientProvider provides real Podman clients
type DefaultClientProvider struct{}

// NewDefaultClientProvider creates a new default client provider
func NewDefaultClientProvider() *DefaultClientProvider {
	return &DefaultClientProvider{}
}

// GetClient returns a real Podman client
func (p *DefaultClientProvider) GetClient() PodmanClient {
	return NewPodmanAdapter()
}

// GetMockClient returns a mock Podman client for testing
func (p *DefaultClientProvider) GetMockClient() *MockPodmanClient {
	return NewMockPodmanClient()
}

// MockClientProvider provides mock Podman clients for testing
type MockClientProvider struct {
	client *MockPodmanClient
}

// NewMockClientProvider creates a new mock client provider
func NewMockClientProvider() *MockClientProvider {
	return &MockClientProvider{
		client: NewMockPodmanClient(),
	}
}

// GetClient returns a mock Podman client
func (p *MockClientProvider) GetClient() PodmanClient {
	return p.client
}

// GetMockClient returns the mock Podman client
func (p *MockClientProvider) GetMockClient() *MockPodmanClient {
	return p.client
}

// ConnectedClient wraps a PodmanClient with automatic connection management
type ConnectedClient struct {
	client    PodmanClient
	connected bool
}

// NewConnectedClient creates a new connected client wrapper
func NewConnectedClient(client PodmanClient) *ConnectedClient {
	return &ConnectedClient{
		client:    client,
		connected: false,
	}
}

// ensureConnected ensures the client is connected
func (c *ConnectedClient) ensureConnected(ctx context.Context) error {
	if !c.connected {
		if err := c.client.Connect(ctx); err != nil {
			return fmt.Errorf("failed to connect to podman: %w", err)
		}
		c.connected = true
	}
	return nil
}

// Close closes the connection and cleans up
func (c *ConnectedClient) Close() error {
	if c.connected {
		err := c.client.Close()
		c.connected = false
		return err
	}
	return nil
}

// GetClient returns the underlying client, ensuring it's connected
func (c *ConnectedClient) GetClient(ctx context.Context) (PodmanClient, error) {
	if err := c.ensureConnected(ctx); err != nil {
		return nil, err
	}
	return c.client, nil
}

// WithClient executes a function with a connected client
func (c *ConnectedClient) WithClient(ctx context.Context, fn func(PodmanClient) error) error {
	client, err := c.GetClient(ctx)
	if err != nil {
		return err
	}
	defer c.Close()
	return fn(client)
}
