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
	client        podman.PodmanClient
	pathManager   *VolumePathManager
	permissionMgr *VolumePermissionManager
	registry      *ManifestRegistry
}

// NewContainerManager creates a new ContainerManager
func NewContainerManager(client podman.PodmanClient) *ContainerManager {
	pathManager := NewVolumePathManager("")
	permissionMgr, err := NewVolumePermissionManager()
	if err != nil {
		// Log error but continue with nil permission manager
		fmt.Printf("Warning: failed to initialize volume permission manager: %v\n", err)
	}

	return &ContainerManager{
		client:        client,
		pathManager:   pathManager,
		permissionMgr: permissionMgr,
	}
}

// NewContainerManagerWithRegistry creates a new ContainerManager with a registry for volume resolution
func NewContainerManagerWithRegistry(client podman.PodmanClient, registry *ManifestRegistry) *ContainerManager {
	pathManager := NewVolumePathManager("")
	permissionMgr, err := NewVolumePermissionManager()
	if err != nil {
		// Log error but continue with nil permission manager
		fmt.Printf("Warning: failed to initialize volume permission manager: %v\n", err)
	}

	return &ContainerManager{
		client:        client,
		pathManager:   pathManager,
		permissionMgr: permissionMgr,
		registry:      registry,
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

	// Validate volume dependencies
	if err := cm.validateVolumeDependencies(container); err != nil {
		return fmt.Errorf("volume dependency validation failed: %w", err)
	}

	// Prepare volume paths and permissions
	if err := cm.prepareVolumeMounts(container); err != nil {
		return fmt.Errorf("failed to prepare volume mounts: %w", err)
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
	spec, err := cm.buildContainerSpec(container)
	if err != nil {
		return fmt.Errorf("unable to build container spec: %w", err)
	}

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
		volumeMount := VolumeMount{
			Name:      mount.Name,
			MountPath: mount.Destination,
			ReadOnly:  !mount.RW,
		}

		// Try to extract subPath from source if it's a bind mount
		if mount.Type == "bind" && mount.Source != "" {
			// For bind mounts, the source might contain subPath information
			// This is a best-effort reconstruction since Podman doesn't store subPath separately
			volumeMount.Name = mount.Name
			if mount.Name == "" {
				// If no name, use the source path as a fallback identifier
				volumeMount.Name = mount.Source
			}
		}

		resource.Spec.Volumes = append(resource.Spec.Volumes, volumeMount)
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

func (cm *ContainerManager) buildContainerSpec(container *ContainerResource) (*specgen.SpecGenerator, error) {
	// Convert volume mounts with enhanced resolution
	mounts, err := cm.convertVolumeMounts(container.Spec.Volumes, container)
	if err != nil {
		return nil, fmt.Errorf("failed to convert volume mounts: %w", err)
	}

	// Process secrets
	env := cm.convertEnvVars(container.Spec.Env)
	secretMounts, err := cm.processSecrets(container.Spec.Secrets)
	if err != nil {
		return nil, fmt.Errorf("failed to process secrets: %w", err)
	}

	spec := &specgen.SpecGenerator{
		ContainerBasicConfig: specgen.ContainerBasicConfig{
			Name:   container.GetName(),
			Env:    env,
			Labels: container.GetLabels(),
		},
		ContainerNetworkConfig: specgen.ContainerNetworkConfig{
			PortMappings: cm.convertPortMappings(container.Spec.Ports),
		},
		ContainerStorageConfig: specgen.ContainerStorageConfig{
			Image:   container.Spec.Image,
			Mounts:  mounts,
			WorkDir: container.Spec.WorkingDir,
			Secrets: secretMounts,
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
	// In Podman, args are combined with command into a single Command field
	if len(container.Spec.Command) > 0 {
		spec.Command = container.Spec.Command
		// Append args to command
		if len(container.Spec.Args) > 0 {
			spec.Command = append(spec.Command, container.Spec.Args...)
		}
	} else if len(container.Spec.Args) > 0 {
		// If only args are specified, use them as the command
		spec.Command = container.Spec.Args
	}

	// Set UID/GID
	if container.Spec.UID != nil {
		spec.User = strconv.FormatInt(*container.Spec.UID, 10)
	}

	// Set restart policy
	if container.Spec.RestartPolicy != "" {
		spec.RestartPolicy = container.Spec.RestartPolicy
	}

	return spec, nil
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

func (cm *ContainerManager) convertVolumeMounts(volumes []VolumeMount, container *ContainerResource) ([]specs.Mount, error) {
	var mounts []specs.Mount

	for _, vol := range volumes {
		// Resolve volume reference to actual volume resource
		volumeResource, err := cm.resolveVolumeReference(vol.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve volume '%s': %w", vol.Name, err)
		}

		// Resolve volume path with subPath support
		pathInfo, err := cm.pathManager.ResolveVolumePath(volumeResource, &vol)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path for volume '%s': %w", vol.Name, err)
		}

		// Determine mount path
		mountPath := vol.MountPath
		if mountPath == "" {
			mountPath = vol.ContainerPath // fallback to legacy field
		}
		if mountPath == "" {
			return nil, fmt.Errorf("mountPath is required for volume '%s'", vol.Name)
		}

		// Build mount options with permission manager
		options, err := cm.buildMountOptions(volumeResource, &vol, container, pathInfo)
		if err != nil {
			return nil, fmt.Errorf("failed to build mount options for volume '%s': %w", vol.Name, err)
		}

		// Create mount specification
		mount := specs.Mount{
			Destination: mountPath,
			Source:      pathInfo.SourcePath,
			Type:        cm.getMountType(volumeResource),
			Options:     options,
		}

		mounts = append(mounts, mount)
	}

	return mounts, nil
}

// validateVolumeDependencies validates that all referenced volumes exist
func (cm *ContainerManager) validateVolumeDependencies(container *ContainerResource) error {
	if cm.registry == nil {
		// If no registry is available, skip validation
		return nil
	}

	for _, vol := range container.Spec.Volumes {
		if vol.Name == "" {
			return fmt.Errorf("volume name cannot be empty")
		}

		// Check if volume exists in registry
		_, exists := cm.registry.GetResource(vol.Name)
		if !exists {
			return fmt.Errorf("referenced volume '%s' does not exist", vol.Name)
		}
	}

	return nil
}

// prepareVolumeMounts prepares volume paths and permissions before container creation
func (cm *ContainerManager) prepareVolumeMounts(container *ContainerResource) error {
	for _, vol := range container.Spec.Volumes {
		// Resolve volume reference
		volumeResource, err := cm.resolveVolumeReference(vol.Name)
		if err != nil {
			return fmt.Errorf("failed to resolve volume '%s': %w", vol.Name, err)
		}

		// Resolve volume path
		pathInfo, err := cm.pathManager.ResolveVolumePath(volumeResource, &vol)
		if err != nil {
			return fmt.Errorf("failed to resolve path for volume '%s': %w", vol.Name, err)
		}

		// Ensure volume path exists
		if err := cm.pathManager.EnsureVolumePath(pathInfo, volumeResource); err != nil {
			return fmt.Errorf("failed to ensure path for volume '%s': %w", vol.Name, err)
		}

		// Manage host directory ownership if needed
		if cm.permissionMgr != nil && volumeResource.Spec.Type == VolumeTypeHostPath {
			if err := cm.permissionMgr.ManageHostDirectoryOwnership(pathInfo.SourcePath, volumeResource); err != nil {
				return fmt.Errorf("failed to manage ownership for volume '%s': %w", vol.Name, err)
			}
		}
	}

	return nil
}

// resolveVolumeReference resolves a volume name to a VolumeResource
func (cm *ContainerManager) resolveVolumeReference(volumeName string) (*VolumeResource, error) {
	if cm.registry == nil {
		return nil, fmt.Errorf("no registry available to resolve volume '%s'", volumeName)
	}

	resource, exists := cm.registry.GetResource(volumeName)
	if !exists {
		return nil, fmt.Errorf("volume '%s' not found in registry", volumeName)
	}

	volumeResource, ok := resource.(*VolumeResource)
	if !ok {
		return nil, fmt.Errorf("resource '%s' is not a volume (type: %s)", volumeName, resource.GetType())
	}

	return volumeResource, nil
}

// buildMountOptions builds Podman mount options for a volume mount
func (cm *ContainerManager) buildMountOptions(volume *VolumeResource, mount *VolumeMount, container *ContainerResource, pathInfo *VolumePathInfo) ([]string, error) {
	var options []string

	// Base mount type options
	switch volume.Spec.Type {
	case VolumeTypeHostPath:
		options = append(options, "bind")
	case VolumeTypeEmptyDir:
		options = append(options, "bind")
	case VolumeTypeVolume:
		// Named volumes don't need bind option
	}

	// Read-only flag
	if mount.ReadOnly {
		options = append(options, "ro")
	} else {
		options = append(options, "rw")
	}

	// Use permission manager to build additional options
	if cm.permissionMgr != nil {
		// Determine if this volume is shared (used by multiple containers)
		sharedAccess := cm.isVolumeShared(volume.GetName())

		permOptions, err := cm.permissionMgr.BuildPodmanMountOptions(volume, mount, sharedAccess)
		if err != nil {
			return nil, fmt.Errorf("failed to build permission options: %w", err)
		}

		// Merge permission options, avoiding duplicates
		for _, opt := range permOptions {
			if !cm.containsOption(options, opt) {
				options = append(options, opt)
			}
		}
	}

	return options, nil
}

// getMountType determines the mount type for a volume
func (cm *ContainerManager) getMountType(volume *VolumeResource) string {
	switch volume.Spec.Type {
	case VolumeTypeHostPath, VolumeTypeEmptyDir:
		return "bind"
	case VolumeTypeVolume:
		return "volume"
	default:
		return "bind"
	}
}

// isVolumeShared checks if a volume is used by multiple containers
func (cm *ContainerManager) isVolumeShared(volumeName string) bool {
	if cm.registry == nil {
		return false
	}

	// Count how many containers reference this volume
	containerCount := 0
	for _, resource := range cm.registry.GetResourcesByType(ResourceTypeContainer) {
		container, ok := resource.(*ContainerResource)
		if !ok {
			continue
		}

		for _, vol := range container.Spec.Volumes {
			if vol.Name == volumeName {
				containerCount++
				if containerCount > 1 {
					return true
				}
				break
			}
		}
	}

	return false
}

// containsOption checks if an option is already in the options slice
func (cm *ContainerManager) containsOption(options []string, option string) bool {
	for _, opt := range options {
		if opt == option {
			return true
		}
	}
	return false
}

// processSecrets processes secret references and returns secret mounts
func (cm *ContainerManager) processSecrets(secrets []SecretReference) ([]specgen.Secret, error) {
	var secretMounts []specgen.Secret

	for _, secretRef := range secrets {
		if secretRef.Env {
			// Mount secret as environment variables
			secretMount := specgen.Secret{
				Source: secretRef.Name,
				Target: "env", // Specify env type for environment variables
				Mode:   0644,
			}
			secretMounts = append(secretMounts, secretMount)
		}

		if secretRef.Path != "" {
			// Mount secret as files
			secretMount := specgen.Secret{
				Source: secretRef.Name,
				Target: secretRef.Path,
				Mode:   0644, // Default file mode
			}
			secretMounts = append(secretMounts, secretMount)
		}
	}

	return secretMounts, nil
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

		// Compare basic fields
		if desiredVol.Name != actualVol.Name || desiredVol.ReadOnly != actualVol.ReadOnly {
			return false
		}

		// Compare subPath
		if desiredVol.SubPath != actualVol.SubPath {
			return false
		}

		// Compare mount options if specified
		if !cm.compareMountOptions(desiredVol.MountOptions, actualVol.MountOptions) {
			return false
		}
	}

	return true
}

// compareMountOptions compares volume mount options
func (cm *ContainerManager) compareMountOptions(desired, actual *VolumeMountOptions) bool {
	// Both nil
	if desired == nil && actual == nil {
		return true
	}

	// One nil, one not
	if desired == nil || actual == nil {
		return false
	}

	// Compare SELinux labels
	if desired.SELinuxLabel != actual.SELinuxLabel {
		return false
	}

	// Compare UID mapping
	if !cm.compareUIDGIDMapping(desired.UIDMapping, actual.UIDMapping) {
		return false
	}

	// Compare GID mapping
	if !cm.compareUIDGIDMapping(desired.GIDMapping, actual.GIDMapping) {
		return false
	}

	return true
}

// compareUIDGIDMapping compares UID/GID mapping configurations
func (cm *ContainerManager) compareUIDGIDMapping(desired, actual *UIDGIDMapping) bool {
	// Both nil
	if desired == nil && actual == nil {
		return true
	}

	// One nil, one not
	if desired == nil || actual == nil {
		return false
	}

	// Compare all fields
	return desired.ContainerID == actual.ContainerID &&
		desired.HostID == actual.HostID &&
		desired.Size == actual.Size
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
