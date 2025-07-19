package podman

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	nettypes "github.com/containers/common/libnetwork/types"
	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/bindings"
	"github.com/containers/podman/v5/pkg/bindings/containers"
	"github.com/containers/podman/v5/pkg/bindings/images"
	"github.com/containers/podman/v5/pkg/bindings/network"
	"github.com/containers/podman/v5/pkg/bindings/secrets"
	"github.com/containers/podman/v5/pkg/bindings/volumes"
	podmantypes "github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/containers/podman/v5/pkg/specgen"
)

// PodmanAdapter implements the PodmanClient interface using Podman bindings
type PodmanAdapter struct {
	ctx context.Context
	uri string
}

// NewPodmanAdapter creates a new PodmanAdapter
func NewPodmanAdapter() *PodmanAdapter {
	uri := getPodmanURI()
	return &PodmanAdapter{
		uri: uri,
	}
}

// Connect establishes a connection to Podman
func (p *PodmanAdapter) Connect(ctx context.Context) error {
	connCtx, err := bindings.NewConnection(ctx, p.uri)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %v", err)
	}
	p.ctx = connCtx
	return nil
}

// Close closes the connection to Podman
func (p *PodmanAdapter) Close() error {
	// Podman bindings don't require explicit cleanup
	return nil
}

// CreateContainer creates a new container
func (p *PodmanAdapter) CreateContainer(ctx context.Context, spec *specgen.SpecGenerator) (*podmantypes.ContainerCreateResponse, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	options := &containers.CreateOptions{}
	response, err := containers.CreateWithSpec(p.ctx, spec, options)
	if err != nil {
		return nil, fmt.Errorf("unable to create container: %v", err)
	}

	return &response, nil
}

// StartContainer starts a container
func (p *PodmanAdapter) StartContainer(ctx context.Context, id string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := containers.Start(p.ctx, id, &containers.StartOptions{})
	if err != nil {
		return fmt.Errorf("unable to start container: %v", err)
	}

	return nil
}

// StopContainer stops a container
func (p *PodmanAdapter) StopContainer(ctx context.Context, name string, timeout uint) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := containers.Stop(p.ctx, name, &containers.StopOptions{Timeout: &timeout})
	if err != nil {
		return fmt.Errorf("unable to stop container: %v", err)
	}

	return nil
}

// RemoveContainer removes a container
func (p *PodmanAdapter) RemoveContainer(ctx context.Context, name string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	_, err := containers.Remove(p.ctx, name, &containers.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove container: %v", err)
	}

	return nil
}

// ListContainers lists containers
func (p *PodmanAdapter) ListContainers(ctx context.Context, filters map[string][]string, all bool) ([]podmantypes.ListContainer, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	list, err := containers.List(p.ctx, &containers.ListOptions{
		All:     &all,
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %v", err)
	}

	return list, nil
}

// InspectContainer inspects a container
func (p *PodmanAdapter) InspectContainer(ctx context.Context, name string) (*define.InspectContainerData, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	inspect, err := containers.Inspect(p.ctx, name, &containers.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container: %v", err)
	}

	return inspect, nil
}

// PullImage pulls an image
func (p *PodmanAdapter) PullImage(ctx context.Context, image string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	options := &images.PullOptions{}
	_, err := images.Pull(p.ctx, image, options)
	if err != nil {
		return fmt.Errorf("unable to pull image: %v", err)
	}

	return nil
}

// GetImage gets image information
func (p *PodmanAdapter) GetImage(ctx context.Context, image string) (*inspect.ImageData, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	imageData, err := images.GetImage(p.ctx, image, &images.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to get image: %v", err)
	}

	return imageData.ImageData, nil
}

// CreateNetwork creates a new network
func (p *PodmanAdapter) CreateNetwork(ctx context.Context, spec NetworkSpec) (*NetworkInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	// Create network configuration
	networkConfig := &nettypes.Network{
		Name:    spec.Name,
		Driver:  spec.Driver,
		Options: spec.Options,
		Labels:  spec.Labels,
	}

	// Set subnet if provided
	if spec.Subnet != "" {
		_, subnet, err := net.ParseCIDR(spec.Subnet)
		if err != nil {
			return nil, fmt.Errorf("invalid subnet format: %v", err)
		}
		networkConfig.Subnets = []nettypes.Subnet{
			{
				Subnet: nettypes.IPNet{IPNet: *subnet},
			},
		}
	}

	response, err := network.Create(p.ctx, networkConfig)
	if err != nil {
		return nil, fmt.Errorf("unable to create network: %v", err)
	}

	return &NetworkInfo{
		ID:      response.ID,
		Name:    response.Name,
		Driver:  response.Driver,
		Options: response.Options,
		Subnet:  spec.Subnet,
		Labels:  response.Labels,
	}, nil
}

// RemoveNetwork removes a network
func (p *PodmanAdapter) RemoveNetwork(ctx context.Context, name string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	_, err := network.Remove(p.ctx, name, &network.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove network: %v", err)
	}

	return nil
}

// ListNetworks lists networks
func (p *PodmanAdapter) ListNetworks(ctx context.Context, filters map[string][]string) ([]NetworkInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	list, err := network.List(p.ctx, &network.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list networks: %v", err)
	}

	var result []NetworkInfo
	for _, net := range list {
		// Extract subnet information
		var subnet string
		if len(net.Subnets) > 0 {
			subnet = net.Subnets[0].Subnet.String()
		}

		result = append(result, NetworkInfo{
			ID:      net.ID,
			Name:    net.Name,
			Driver:  net.Driver,
			Options: net.Options,
			Subnet:  subnet,
			Labels:  net.Labels,
		})
	}

	return result, nil
}

// InspectNetwork inspects a network
func (p *PodmanAdapter) InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	inspect, err := network.Inspect(p.ctx, name, &network.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to inspect network: %v", err)
	}

	// Extract subnet information
	var subnet string
	if len(inspect.Subnets) > 0 {
		subnet = inspect.Subnets[0].Subnet.String()
	}

	return &NetworkInfo{
		ID:      inspect.ID,
		Name:    inspect.Name,
		Driver:  inspect.Driver,
		Options: inspect.Options,
		Subnet:  subnet,
		Labels:  inspect.Labels,
	}, nil
}

// ConnectContainerToNetwork connects a container to a network
func (p *PodmanAdapter) ConnectContainerToNetwork(ctx context.Context, containerName, networkName string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := network.Connect(p.ctx, networkName, containerName, nil)
	if err != nil {
		return fmt.Errorf("unable to connect container to network: %v", err)
	}

	return nil
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (p *PodmanAdapter) DisconnectContainerFromNetwork(ctx context.Context, containerName, networkName string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := network.Disconnect(p.ctx, networkName, containerName, nil)
	if err != nil {
		return fmt.Errorf("unable to disconnect container from network: %v", err)
	}

	return nil
}

// CreateVolume creates a new volume
func (p *PodmanAdapter) CreateVolume(ctx context.Context, spec VolumeSpec) (*VolumeInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	createOptions := podmantypes.VolumeCreateOptions{
		Name:    spec.Name,
		Driver:  spec.Driver,
		Options: spec.Options,
		Labels:  spec.Labels,
	}

	options := &volumes.CreateOptions{}
	response, err := volumes.Create(p.ctx, createOptions, options)
	if err != nil {
		return nil, fmt.Errorf("unable to create volume: %v", err)
	}

	return &VolumeInfo{
		Name:       response.Name,
		Driver:     response.Driver,
		Mountpoint: response.Mountpoint,
		Options:    response.Options,
		Labels:     response.Labels,
	}, nil
}

// RemoveVolume removes a volume
func (p *PodmanAdapter) RemoveVolume(ctx context.Context, name string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := volumes.Remove(p.ctx, name, &volumes.RemoveOptions{})
	if err != nil {
		return fmt.Errorf("unable to remove volume: %v", err)
	}

	return nil
}

// ListVolumes lists volumes
func (p *PodmanAdapter) ListVolumes(ctx context.Context, filters map[string][]string) ([]VolumeInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	list, err := volumes.List(p.ctx, &volumes.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list volumes: %v", err)
	}

	var result []VolumeInfo
	for _, vol := range list {
		result = append(result, VolumeInfo{
			Name:       vol.Name,
			Driver:     vol.Driver,
			Mountpoint: vol.Mountpoint,
			Options:    vol.Options,
			Labels:     vol.Labels,
		})
	}

	return result, nil
}

// InspectVolume inspects a volume
func (p *PodmanAdapter) InspectVolume(ctx context.Context, name string) (*VolumeInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	inspect, err := volumes.Inspect(p.ctx, name, &volumes.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to inspect volume: %v", err)
	}

	return &VolumeInfo{
		Name:       inspect.Name,
		Driver:     inspect.Driver,
		Mountpoint: inspect.Mountpoint,
		Options:    inspect.Options,
		Labels:     inspect.Labels,
	}, nil
}

// CreateSecret creates a new secret
func (p *PodmanAdapter) CreateSecret(ctx context.Context, spec SecretSpec) (*SecretInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	reader := strings.NewReader(string(spec.Data))
	options := &secrets.CreateOptions{
		Name:   &spec.Name,
		Labels: spec.Labels,
	}

	response, err := secrets.Create(p.ctx, reader, options)
	if err != nil {
		return nil, fmt.Errorf("unable to create secret: %v", err)
	}

	return &SecretInfo{
		ID:     response.ID,
		Name:   spec.Name,
		Labels: spec.Labels,
	}, nil
}

// UpdateSecret updates an existing secret
func (p *PodmanAdapter) UpdateSecret(ctx context.Context, name string, spec SecretSpec) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	// Podman doesn't support updating secrets directly, so we need to remove and recreate
	// This is a limitation of Podman's secret implementation
	err := p.RemoveSecret(ctx, name)
	if err != nil {
		return fmt.Errorf("unable to remove existing secret for update: %v", err)
	}

	_, err = p.CreateSecret(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to recreate secret: %v", err)
	}

	return nil
}

// RemoveSecret removes a secret
func (p *PodmanAdapter) RemoveSecret(ctx context.Context, name string) error {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return err
		}
	}

	err := secrets.Remove(p.ctx, name)
	if err != nil {
		return fmt.Errorf("unable to remove secret: %v", err)
	}

	return nil
}

// ListSecrets lists secrets
func (p *PodmanAdapter) ListSecrets(ctx context.Context, filters map[string][]string) ([]SecretInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	list, err := secrets.List(p.ctx, &secrets.ListOptions{
		Filters: filters,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to list secrets: %v", err)
	}

	var result []SecretInfo
	for _, secret := range list {
		result = append(result, SecretInfo{
			ID:     secret.ID,
			Name:   secret.Spec.Name,
			Labels: secret.Spec.Labels,
		})
	}

	return result, nil
}

// InspectSecret inspects a secret
func (p *PodmanAdapter) InspectSecret(ctx context.Context, name string) (*SecretInfo, error) {
	if p.ctx == nil {
		if err := p.Connect(ctx); err != nil {
			return nil, err
		}
	}

	inspect, err := secrets.Inspect(p.ctx, name, &secrets.InspectOptions{})
	if err != nil {
		return nil, fmt.Errorf("unable to inspect secret: %v", err)
	}

	return &SecretInfo{
		ID:     inspect.ID,
		Name:   inspect.Spec.Name,
		Labels: inspect.Spec.Labels,
	}, nil
}

// getPodmanURI returns the Podman socket URI
func getPodmanURI() string {
	if env, exists := os.LookupEnv("PODMAN_SOCK"); exists {
		return env
	}
	return "unix:/run/user/1000/podman/podman.sock"
}
