# Podman Client Abstraction

This package implements the abstracted Podman client interface as specified in task 1.2 of the core resource reconciliation specification.

## Overview

The abstraction provides a clean interface for interacting with Podman resources while supporting both real Podman instances and mock implementations for testing.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Client Providers                         │
├─────────────────────────────────────────────────────────────┤
│                    PodmanClient Interface                   │
├─────────────────────────────────────────────────────────────┤
│  PodmanAdapter (Real)          MockPodmanClient (Testing)   │
├─────────────────────────────────────────────────────────────┤
│              Podman Bindings              Mock Storage      │
└─────────────────────────────────────────────────────────────┘
```

## Components

### PodmanClient Interface (`client.go`)
- Defines the contract for all Podman operations
- Supports containers, networks, volumes, secrets, and images
- Provides consistent error handling and context support

### PodmanAdapter (`adapter.go`)
- Implements PodmanClient using real Podman bindings
- Wraps the Podman v5 bindings library
- Handles connection management and error translation

### MockPodmanClient (`mock.go`)
- Full mock implementation for testing
- Supports all PodmanClient operations
- Includes configurable failure modes and call tracking
- Thread-safe with proper synchronization

### Client Providers (`provider.go`)
- Factory pattern for creating clients
- DefaultClientProvider: Returns real PodmanAdapter
- MockClientProvider: Returns MockPodmanClient
- Enables dependency injection and testing

### ConnectedClient Wrapper (`provider.go`)
- Automatic connection management
- WithClient pattern for resource cleanup
- Ensures connections are properly closed

## Usage Examples

### Basic Usage with Mock Client
```go
mockClient := NewMockPodmanClient()
ctx := context.Background()

// Connect and create container
err := mockClient.Connect(ctx)
if err != nil {
    return err
}
defer mockClient.Close()

spec := &specgen.SpecGenerator{
    ContainerBasicConfig: specgen.ContainerBasicConfig{
        Name: "test-container",
    },
    ContainerStorageConfig: specgen.ContainerStorageConfig{
        Image: "nginx:latest",
    },
}

response, err := mockClient.CreateContainer(ctx, spec)
```

### Using Client Providers
```go
// For testing
mockProvider := NewMockClientProvider()
client := mockProvider.GetClient()

// For production
defaultProvider := NewDefaultClientProvider()
client := defaultProvider.GetClient()
```

### Automatic Connection Management
```go
mockClient := NewMockPodmanClient()
connectedClient := NewConnectedClient(mockClient)

err := connectedClient.WithClient(ctx, func(client PodmanClient) error {
    // Client is automatically connected and cleaned up
    return client.PullImage(ctx, "nginx:latest")
})
```

## Supported Operations

### Container Operations
- CreateContainer
- StartContainer
- StopContainer
- RemoveContainer
- ListContainers
- InspectContainer

### Network Operations
- CreateNetwork
- RemoveNetwork
- ListNetworks
- InspectNetwork
- ConnectContainerToNetwork
- DisconnectContainerFromNetwork

### Volume Operations
- CreateVolume
- RemoveVolume
- ListVolumes
- InspectVolume

### Secret Operations
- CreateSecret
- UpdateSecret
- RemoveSecret
- ListSecrets
- InspectSecret

### Image Operations
- PullImage
- GetImage

## Testing Features

### Mock Client Capabilities
- **State Management**: Maintains in-memory state for all resources
- **Call Tracking**: Records method calls for verification
- **Error Simulation**: Configurable failure modes for testing error handling
- **Filter Support**: Implements label-based filtering like real Podman
- **Thread Safety**: Safe for concurrent use

### Test Helpers
```go
mockClient := NewMockPodmanClient()

// Configure failures
mockClient.SetShouldFailConnect(true)
mockClient.SetShouldFailOperation("CreateContainer", true)

// Verify calls
assert.Equal(t, 1, mockClient.GetCallCount("CreateContainer"))

// Reset state
mockClient.Reset()
```

## Integration with Existing Code

The abstraction is designed to work seamlessly with existing container code:

```go
// Existing function signature remains unchanged
func RemoveContainer(ctx context.Context, client podman.PodmanClient, name string) error {
    // Works with both PodmanAdapter and MockPodmanClient
    if err := client.Connect(ctx); err != nil {
        return err
    }
    defer client.Close()
    
    // ... rest of implementation
}
```

## Task Completion Status

✅ **Extract existing Podman bindings into PodmanClient interface**
- Complete interface definition with all necessary operations
- Consistent method signatures across all resource types

✅ **Create adapter pattern for container, network, volume, secret operations**
- PodmanAdapter implements the interface using real Podman bindings
- Proper error handling and connection management
- Support for all resource types

✅ **Implement mock client for testing and development**
- Full MockPodmanClient implementation
- Comprehensive test coverage
- Configurable behavior for various testing scenarios

## Future Enhancements

- Network operations in PodmanAdapter need proper implementation once network bindings are available
- Additional error types and recovery mechanisms
- Performance optimizations for bulk operations
- Connection pooling for high-throughput scenarios

## Files

- `client.go` - PodmanClient interface and type definitions
- `adapter.go` - Real Podman implementation
- `mock.go` - Mock implementation for testing
- `provider.go` - Client factory and connection management
- `*_test.go` - Comprehensive test suite
- `example_usage.go` - Usage examples and demonstrations