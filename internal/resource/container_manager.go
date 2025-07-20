package resource

import (
	"context"
	"cutepod/internal/podman"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	nettypes "github.com/containers/common/libnetwork/types"
	podmantypes "github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/specgen"
	"github.com/opencontainers/runtime-spec/specs-go"
)

// ContainerManager implements ResourceManager for container resources
type ContainerManager struct {
	client podman.PodmanClient
}

// NewContainerManager creates a new ContainerManager
func NewContainerManager(client podman.PodmanClient) *ContainerManager {
	return &ContainerManager{
		client: client,
	}
}

// GetResourceType returns the resource type this manager handles
func (cm *ContainerManager) GetResourceType() ResourceType {
	return ResourceTypeContainer
}

// GetDesiredState extracts container resources from manifests
func (cm *ContainerManager) GetDesiredState(manifests []Resource) ([]Resource, error) {
	var containers []Resource

	for _, manifest := range manifests {
		if manifest.GetType() == ResourceTypeContainer {
			containers = append(containers, manifest)
		}
	}

	return containers, nil
}

// GetActualState retrieves current container resources from Podman
func (cm *ContainerManager) GetActualState(ctx context.Context, chartName string) ([]Resource, error) {
	connectedClient := podman.NewConnectedClient(cm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to podman: %w", err)
	}

	containers, err := podmanClient.ListContainers(
		ctx,
		map[string][]string{
			"label": {"cutepod.io/chart=" + chartName},
		},
		true,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to list containers: %w", err)
	}

	var resources []Resource
	for _, container := range containers {
		// Convert Podman container to ContainerResource
		resource, err := cm.convertPodmanContainerToResource(ctx, podmanClient, container)
		if err != nil {
			return nil, fmt.Errorf("unable to convert container %s: %w", container.Names[0], err)
		}
		resources = append(resources, resource)
	}

	return resources, nil
}

// CreateResource creates a new container resource
func (cm *ContainerManager) CreateResource(ctx context.Context, resource Resource) error {
	container, ok := resource.(*ContainerResource)
	if !ok {
		return fmt.Errorf("expected ContainerResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(cm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	// Pull image if needed
	if err := cm.pullImageIfNeeded(ctx, podmanClient, container.Spec.Image); err != nil {
		return fmt.Errorf("unable to pull image: %w", err)
	}

	// Create container spec
	spec := cm.buildContainerSpec(container)

	// Create container
	response, err := podmanClient.CreateContainer(ctx, spec)
	if err != nil {
		return fmt.Errorf("unable to create container: %w", err)
	}

	// Start container
	if err := podmanClient.StartContainer(ctx, response.ID); err != nil {
		return fmt.Errorf("unable to start container: %w", err)
	}

	return nil
}

// UpdateResource updates an existing container resource
func (cm *ContainerManager) UpdateResource(ctx context.Context, desired, actual Resource) error {
	// For containers, update typically means recreate
	// First remove the existing container, then create the new one
	if err := cm.DeleteResource(ctx, actual); err != nil {
		return fmt.Errorf("unable to remove existing container for update: %w", err)
	}

	if err := cm.CreateResource(ctx, desired); err != nil {
		return fmt.Errorf("unable to create updated container: %w", err)
	}

	return nil
}

// DeleteResource deletes a container resource
func (cm *ContainerManager) DeleteResource(ctx context.Context, resource Resource) error {
	container, ok := resource.(*ContainerResource)
	if !ok {
		return fmt.Errorf("expected ContainerResource, got %T", resource)
	}

	connectedClient := podman.NewConnectedClient(cm.client)
	defer connectedClient.Close()

	podmanClient, err := connectedClient.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("unable to connect to podman: %w", err)
	}

	return cm.removeContainer(ctx, podmanClient, container.GetName())
}

// CompareResources compares desired vs actual container resource
func (cm *ContainerManager) CompareResources(desired, actual Resource) (bool, error) {
	desiredContainer, ok := desired.(*ContainerResource)
	if !ok {
		return false, fmt.Errorf("expected ContainerResource for desired, got %T", desired)
	}

	actualContainer, ok := actual.(*ContainerResource)
	if !ok {
		return false, fmt.Errorf("expected ContainerResource for actual, got %T", actual)
	}

	// Compare key fields that would require recreation
	if desiredContainer.Spec.Image != actualContainer.Spec.Image {
		return false, nil
	}

	if !slices.Equal(desiredContainer.Spec.Command, actualContainer.Spec.Command) {
		return false, nil
	}

	if !slices.Equal(desiredContainer.Spec.Args, actualContainer.Spec.Args) {
		return false, nil
	}

	if desiredContainer.Spec.WorkingDir != actualContainer.Spec.WorkingDir {
		return false, nil
	}

	// Compare environment variables
	if !cm.compareEnvVars(desiredContainer.Spec.Env, actualContainer.Spec.Env) {
		return false, nil
	}

	// Compare ports
	if !cm.comparePorts(desiredContainer.Spec.Ports, actualContainer.Spec.Ports) {
		return false, nil
	}

	// Compare volumes
	if !cm.compareVolumes(desiredContainer.Spec.Volumes, actualContainer.Spec.Volumes) {
		return false, nil
	}

	// Compare networks
	if !slices.Equal(desiredContainer.Spec.Networks, actualContainer.Spec.Networks) {
		return false, nil
	}

	// Compare secrets
	if !cm.compareSecrets(desiredContainer.Spec.Secrets, actualContainer.Spec.Secrets) {
		return false, nil
	}

	// Compare restart policy
	if desiredContainer.Spec.RestartPolicy != actualContainer.Spec.RestartPolicy {
		return false, nil
	}

	return true, nil
}

// Helper methods

func (cm *ContainerManager) convertPodmanContainerToResource(ctx context.Context, client podman.PodmanClient, container podmantypes.ListContainer) (*ContainerResource, error) {
	// Get detailed container information
	inspect, err := client.InspectContainer(ctx, container.Names[0])
	if err != nil {
		return nil, fmt.Errorf("unable to inspect container: %w", err)
	}

	resource := NewContainerResource()
	resource.ObjectMeta.Name = strings.TrimPrefix(container.Names[0], "/")
	resource.SetLabels(container.Labels)

	// Convert inspect data to ContainerResource spec
	if inspect.Config != nil {
		resource.Spec.Image = inspect.Config.Image
		resource.Spec.Command = inspect.Config.Cmd
		resource.Spec.WorkingDir = inspect.Config.WorkingDir
	}
	resource.Spec.Args = inspect.Args

	// Convert environment variables
	if inspect.Config != nil && inspect.Config.Env != nil {
		for _, env := range inspect.Config.Env {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				resource.Spec.Env = append(resource.Spec.Env, EnvVar{
					Name:  parts[0],
					Value: parts[1],
				})
			}
		}
	}

	// Convert ports
	if inspect.HostConfig != nil && inspect.HostConfig.PortBindings != nil {
		for portProto, bindings := range inspect.HostConfig.PortBindings {
			parts := strings.Split(portProto, "/")
			if len(parts) != 2 {
				continue
			}
			containerPort, err := strconv.ParseUint(parts[0], 10, 16)
			if err != nil {
				continue
			}
			protocol := strings.ToUpper(parts[1])

			for _, binding := range bindings {
				hostPort, err := strconv.ParseUint(binding.HostPort, 10, 16)
				if err != nil {
					continue
				}
				resource.Spec.Ports = append(resource.Spec.Ports, ContainerPort{
					ContainerPort: uint16(containerPort),
					HostPort:      uint16(hostPort),
					Protocol:      protocol,
				})
			}
		}
	}

	// Convert volumes
	for _, mount := range inspect.Mounts {
		resource.Spec.Volumes = append(resource.Spec.Volumes, VolumeMount{
			Name:      mount.Name,
			MountPath: mount.Destination,
			ReadOnly:  !mount.RW,
		})
	}

	// Convert restart policy
	if inspect.HostConfig != nil && inspect.HostConfig.RestartPolicy != nil {
		resource.Spec.RestartPolicy = inspect.HostConfig.RestartPolicy.Name
	}

	return resource, nil
}

func (cm *ContainerManager) pullImageIfNeeded(ctx context.Context, client podman.PodmanClient, image string) error {
	existingImage, err := client.GetImage(ctx, image)
	if err == nil && existingImage != nil {
		return nil
	}

	return client.PullImage(ctx, image)
}

func (cm *ContainerManager) buildContainerSpec(container *ContainerResource) *specgen.SpecGenerator {
	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:   container.GetName(),
			Env:    cm.convertEnvVars(container.Spec.Env),
			Labels: container.GetLabels(),
		},
		ContainerNetworkConfig: specgen.ContainerNetworkConfig{
			PortMappings: cm.convertPortMappings(container.Spec.Ports),
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image:   container.Spec.Image,
			Mounts:  cm.convertVolumeMounts(container.Spec.Volumes),
			WorkDir: container.Spec.WorkingDir,
		},
		ContainerHealthCheckConfig: specgen.ContainerHealthCheckConfig{
			HealthLogDestination: "/tmp",
		},
		ContainerSecurityConfig: specgen.ContainerSecurityConfig{
			Privileged: cm.getPrivileged(container.Spec.SecurityContext),
			CapAdd:     cm.getCapabilities(container.Spec.SecurityContext, true),
			CapDrop:    cm.getCapabilities(container.Spec.SecurityContext, false),
		},
	}

	// Set command and args
	if len(container.Spec.Command) > 0 {
		spec.Command = container.Spec.Command
	}
	// Note: Args are typically handled as part of Command in Podman

	// Set UID/GID
	if container.Spec.UID != nil {
		spec.User = strconv.FormatInt(*container.Spec.UID, 10)
	}

	// Set restart policy
	if container.Spec.RestartPolicy != "" {
		spec.RestartPolicy = container.Spec.RestartPolicy
	}

	return spec
}

func (cm *ContainerManager) convertEnvVars(envVars []EnvVar) map[string]string {
	env := make(map[string]string)
	for _, e := range envVars {
		env[e.Name] = e.Value
	}
	return env
}

func (cm *ContainerManager) convertPortMappings(ports []ContainerPort) []nettypes.PortMapping {
	var mappings []nettypes.PortMapping
	for _, port := range ports {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		mappings = append(mappings, nettypes.PortMapping{
			HostPort:      port.HostPort,
			ContainerPort: port.ContainerPort,
			Protocol:      strings.ToLower(protocol),
		})
	}
	return mappings
}

func (cm *ContainerManager) convertVolumeMounts(volumes []VolumeMount) []specs.Mount {
	var mounts []specs.Mount
	for _, vol := range volumes {
		mountPath := vol.MountPath
		if mountPath == "" {
			mountPath = vol.ContainerPath // fallback to legacy field
		}

		// TODO: Resolve volume name to actual source path
		// This will be enhanced when volume managers are implemented
		source := vol.Name
		if source == "" {
			source = "/tmp/cutepod-volumes/" + vol.Name
		}

		mounts = append(mounts, specs.Mount{
			Destination: mountPath,
			Source:      source,
			Options:     cm.getMountOptions(vol.ReadOnly),
		})
	}
	return mounts
}

func (cm *ContainerManager) getMountOptions(readOnly bool) []string {
	if readOnly {
		return []string{"ro"}
	}
	return []string{"rw"}
}

func (cm *ContainerManager) getPrivileged(secCtx *SecurityContext) *bool {
	if secCtx != nil && secCtx.Privileged != nil {
		return secCtx.Privileged
	}
	return nil
}

func (cm *ContainerManager) getCapabilities(secCtx *SecurityContext, add bool) []string {
	if secCtx == nil || secCtx.Capabilities == nil {
		return nil
	}
	if add {
		return secCtx.Capabilities.Add
	}
	return secCtx.Capabilities.Drop
}

func (cm *ContainerManager) removeContainer(ctx context.Context, client podman.PodmanClient, name string) error {
	timeout, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	// Stop container first
	if err := client.StopContainer(timeout, name, 15); err != nil {
		// Continue with removal even if stop fails
		fmt.Printf("Warning: failed to stop container %s: %v\n", name, err)
	}

	// Remove container
	if err := client.RemoveContainer(ctx, name); err != nil {
		return fmt.Errorf("unable to remove container %s: %w", name, err)
	}

	return nil
}

// Comparison helper methods

func (cm *ContainerManager) compareEnvVars(desired, actual []EnvVar) bool {
	if len(desired) != len(actual) {
		return false
	}

	desiredMap := make(map[string]string)
	for _, env := range desired {
		desiredMap[env.Name] = env.Value
	}

	actualMap := make(map[string]string)
	for _, env := range actual {
		actualMap[env.Name] = env.Value
	}

	for k, v := range desiredMap {
		if actualMap[k] != v {
			return false
		}
	}

	return true
}

func (cm *ContainerManager) comparePorts(desired, actual []ContainerPort) bool {
	if len(desired) != len(actual) {
		return false
	}

	// Create maps for comparison
	desiredMap := make(map[string]uint16)
	for _, port := range desired {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		key := fmt.Sprintf("%d/%s", port.ContainerPort, strings.ToLower(protocol))
		desiredMap[key] = port.HostPort
	}

	actualMap := make(map[string]uint16)
	for _, port := range actual {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		key := fmt.Sprintf("%d/%s", port.ContainerPort, strings.ToLower(protocol))
		actualMap[key] = port.HostPort
	}

	for k, v := range desiredMap {
		if actualMap[k] != v {
			return false
		}
	}

	return true
}

func (cm *ContainerManager) compareVolumes(desired, actual []VolumeMount) bool {
	if len(desired) != len(actual) {
		return false
	}

	// Create maps for comparison
	desiredMap := make(map[string]VolumeMount)
	for _, vol := range desired {
		mountPath := vol.MountPath
		if mountPath == "" {
			mountPath = vol.ContainerPath
		}
		desiredMap[mountPath] = vol
	}

	actualMap := make(map[string]VolumeMount)
	for _, vol := range actual {
		mountPath := vol.MountPath
		if mountPath == "" {
			mountPath = vol.ContainerPath
		}
		actualMap[mountPath] = vol
	}

	for path, desiredVol := range desiredMap {
		actualVol, exists := actualMap[path]
		if !exists {
			return false
		}
		if desiredVol.Name != actualVol.Name || desiredVol.ReadOnly != actualVol.ReadOnly {
			return false
		}
	}

	return true
}

func (cm *ContainerManager) compareSecrets(desired, actual []SecretReference) bool {
	if len(desired) != len(actual) {
		return false
	}

	// Create maps for comparison
	desiredMap := make(map[string]SecretReference)
	for _, secret := range desired {
		desiredMap[secret.Name] = secret
	}

	actualMap := make(map[string]SecretReference)
	for _, secret := range actual {
		actualMap[secret.Name] = secret
	}

	for name, desiredSecret := range desiredMap {
		actualSecret, exists := actualMap[name]
		if !exists {
			return false
		}
		if desiredSecret.Env != actualSecret.Env || desiredSecret.Path != actualSecret.Path {
			return false
		}
	}

	return true
}

// GetPodmanURI returns the Podman socket URI
func GetPodmanURI() string {
	if env, exists := os.LookupEnv("PODMAN_SOCK"); exists {
		return env
	}
	return "unix:/run/user/1000/podman/podman.sock"
}
