package podman

import (
	"context"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/containers/podman/v5/pkg/specgen"
)

// PodmanClient defines the interface for interacting with Podman
type PodmanClient interface {
	// Container operations
	CreateContainer(ctx context.Context, spec *specgen.SpecGenerator) (*types.ContainerCreateResponse, error)
	StartContainer(ctx context.Context, id string) error
	StopContainer(ctx context.Context, name string, timeout uint) error
	RemoveContainer(ctx context.Context, name string) error
	ListContainers(ctx context.Context, filters map[string][]string, all bool) ([]types.ListContainer, error)
	InspectContainer(ctx context.Context, name string) (*define.InspectContainerData, error)
	
	// Network operations
	CreateNetwork(ctx context.Context, spec NetworkSpec) (*NetworkInfo, error)
	RemoveNetwork(ctx context.Context, name string) error
	ListNetworks(ctx context.Context, filters map[string][]string) ([]NetworkInfo, error)
	InspectNetwork(ctx context.Context, name string) (*NetworkInfo, error)
	ConnectContainerToNetwork(ctx context.Context, containerName, networkName string) error
	DisconnectContainerFromNetwork(ctx context.Context, containerName, networkName string) error
	
	// Volume operations
	CreateVolume(ctx context.Context, spec VolumeSpec) (*VolumeInfo, error)
	RemoveVolume(ctx context.Context, name string) error
	ListVolumes(ctx context.Context, filters map[string][]string) ([]VolumeInfo, error)
	InspectVolume(ctx context.Context, name string) (*VolumeInfo, error)
	
	// Secret operations
	CreateSecret(ctx context.Context, spec SecretSpec) (*SecretInfo, error)
	UpdateSecret(ctx context.Context, name string, spec SecretSpec) error
	RemoveSecret(ctx context.Context, name string) error
	ListSecrets(ctx context.Context, filters map[string][]string) ([]SecretInfo, error)
	InspectSecret(ctx context.Context, name string) (*SecretInfo, error)
	
	// Image operations
	PullImage(ctx context.Context, image string) error
	GetImage(ctx context.Context, image string) (*inspect.ImageData, error)
	
	// Connection management
	Connect(ctx context.Context) error
	Close() error
}

// ContainerSpec represents the specification for creating a container
type ContainerSpec struct {
	Name         string
	Image        string
	Command      []string
	Args         []string
	Env          map[string]string
	Ports        []PortMapping
	Volumes      []VolumeMount
	WorkingDir   string
	Labels       map[string]string
	Capabilities *Capabilities
	Privileged   bool
	Resources    *ResourceLimits
}

// PortMapping represents a port mapping configuration
type PortMapping struct {
	HostPort      uint16
	ContainerPort uint16
	Protocol      string
}

// VolumeMount represents a volume mount configuration
type VolumeMount struct {
	Source      string
	Destination string
	ReadOnly    bool
}

// Capabilities represents security capabilities
type Capabilities struct {
	Add  []string
	Drop []string
}

// ResourceLimits represents resource constraints
type ResourceLimits struct {
	Memory   int64
	NanoCPUs int64
}

// NetworkSpec represents the specification for creating a network
type NetworkSpec struct {
	Name    string
	Driver  string
	Options map[string]string
	Subnet  string
	Labels  map[string]string
}

// NetworkInfo represents network information
type NetworkInfo struct {
	ID      string
	Name    string
	Driver  string
	Options map[string]string
	Subnet  string
	Labels  map[string]string
}

// VolumeSpec represents the specification for creating a volume
type VolumeSpec struct {
	Name     string
	Driver   string
	Options  map[string]string
	Labels   map[string]string
	HostPath string // for bind mounts
}

// VolumeInfo represents volume information
type VolumeInfo struct {
	Name       string
	Driver     string
	Mountpoint string
	Options    map[string]string
	Labels     map[string]string
}

// SecretSpec represents the specification for creating a secret
type SecretSpec struct {
	Name   string
	Data   []byte
	Labels map[string]string
}

// SecretInfo represents secret information
type SecretInfo struct {
	ID     string
	Name   string
	Labels map[string]string
}