package resource

import (
	"testing"

	"cutepod/internal/podman"
)

func TestContainerManager_EnhancedVolumeIntegration(t *testing.T) {
	// Create a mock Podman client
	mockClient := &podman.MockPodmanClient{}

	// Create a registry with volume resources
	registry := NewManifestRegistry()

	// Add a hostPath volume
	hostPathVolume := NewVolumeResource()
	hostPathVolume.ObjectMeta.Name = "test-hostpath"
	hostPathVolume.Spec.Type = VolumeTypeHostPath
	hostPathVolume.Spec.HostPath = &HostPathVolumeSource{
		Path: "/tmp/test-data",
		Type: &[]HostPathType{HostPathDirectoryOrCreate}[0],
	}
	hostPathVolume.Spec.SecurityContext = &VolumeSecurityContext{
		Owner: &VolumeOwnership{
			User:  &[]int64{1000}[0],
			Group: &[]int64{1000}[0],
		},
		SELinuxOptions: &SELinuxVolumeOptions{
			Level: "shared",
		},
	}

	err := registry.AddResource(hostPathVolume)
	if err != nil {
		t.Fatalf("Failed to add hostPath volume to registry: %v", err)
	}

	// Add an emptyDir volume
	emptyDirVolume := NewVolumeResource()
	emptyDirVolume.ObjectMeta.Name = "test-emptydir"
	emptyDirVolume.Spec.Type = VolumeTypeEmptyDir
	emptyDirVolume.Spec.EmptyDir = &EmptyDirVolumeSource{
		SizeLimit: &[]string{"1Gi"}[0],
	}

	err = registry.AddResource(emptyDirVolume)
	if err != nil {
		t.Fatalf("Failed to add emptyDir volume to registry: %v", err)
	}

	// Create container manager with registry
	cm := NewContainerManagerWithRegistry(mockClient, registry)

	// Create a container that uses both volumes
	container := NewContainerResource()
	container.ObjectMeta.Name = "test-container"
	container.Spec.Image = "nginx:latest"
	container.Spec.UID = &[]int64{1000}[0]
	container.Spec.GID = &[]int64{1000}[0]
	container.Spec.Volumes = []VolumeMount{
		{
			Name:      "test-hostpath",
			MountPath: "/usr/share/nginx/html",
			SubPath:   "web-content",
			ReadOnly:  true,
			MountOptions: &VolumeMountOptions{
				SELinuxLabel: "z",
			},
		},
		{
			Name:      "test-emptydir",
			MountPath: "/var/cache/nginx",
			SubPath:   "cache",
			ReadOnly:  false,
		},
		{
			Name:      "test-hostpath",
			MountPath: "/etc/nginx/nginx.conf",
			SubPath:   "configs/nginx.conf",
			ReadOnly:  true,
		},
	}

	// Test volume dependency validation
	t.Run("ValidateVolumeDependencies", func(t *testing.T) {
		err := cm.validateVolumeDependencies(container)
		if err != nil {
			t.Errorf("Volume dependency validation failed: %v", err)
		}
	})

	// Test volume mount conversion
	t.Run("ConvertVolumeMounts", func(t *testing.T) {
		mounts, err := cm.convertVolumeMounts(container.Spec.Volumes, container)
		if err != nil {
			t.Errorf("Volume mount conversion failed: %v", err)
			return
		}

		if len(mounts) != 3 {
			t.Errorf("Expected 3 mounts, got %d", len(mounts))
			return
		}

		// Check first mount (hostPath with subPath)
		mount1 := mounts[0]
		if mount1.Destination != "/usr/share/nginx/html" {
			t.Errorf("Expected destination '/usr/share/nginx/html', got '%s'", mount1.Destination)
		}
		if mount1.Type != "bind" {
			t.Errorf("Expected type 'bind', got '%s'", mount1.Type)
		}
		if !containsString(mount1.Options, "ro") {
			t.Errorf("Expected 'ro' option for read-only mount")
		}
		if !containsString(mount1.Options, "z") {
			t.Errorf("Expected 'z' SELinux option")
		}

		// Check second mount (emptyDir with subPath)
		mount2 := mounts[1]
		if mount2.Destination != "/var/cache/nginx" {
			t.Errorf("Expected destination '/var/cache/nginx', got '%s'", mount2.Destination)
		}
		if mount2.Type != "bind" {
			t.Errorf("Expected type 'bind', got '%s'", mount2.Type)
		}
		if !containsString(mount2.Options, "rw") {
			t.Errorf("Expected 'rw' option for read-write mount")
		}

		// Check third mount (file mount via subPath)
		mount3 := mounts[2]
		if mount3.Destination != "/etc/nginx/nginx.conf" {
			t.Errorf("Expected destination '/etc/nginx/nginx.conf', got '%s'", mount3.Destination)
		}
	})

	// Test container spec building
	t.Run("BuildContainerSpec", func(t *testing.T) {
		spec, err := cm.buildContainerSpec(container)
		if err != nil {
			t.Errorf("Container spec building failed: %v", err)
			return
		}

		if spec.Image != "nginx:latest" {
			t.Errorf("Expected image 'nginx:latest', got '%s'", spec.Image)
		}

		if len(spec.Mounts) != 3 {
			t.Errorf("Expected 3 mounts in spec, got %d", len(spec.Mounts))
		}
	})

	// Test volume dependency validation with missing volume
	t.Run("ValidateVolumeDependencies_MissingVolume", func(t *testing.T) {
		containerWithMissingVol := NewContainerResource()
		containerWithMissingVol.ObjectMeta.Name = "test-container-missing"
		containerWithMissingVol.Spec.Image = "nginx:latest"
		containerWithMissingVol.Spec.Volumes = []VolumeMount{
			{
				Name:      "non-existent-volume",
				MountPath: "/data",
			},
		}

		err := cm.validateVolumeDependencies(containerWithMissingVol)
		if err == nil {
			t.Error("Expected error for missing volume dependency, got nil")
		}
	})
}

func TestContainerManager_VolumeComparison(t *testing.T) {
	cm := NewContainerManager(&podman.MockPodmanClient{})

	// Test volume mount comparison with enhanced fields
	t.Run("CompareVolumes_Enhanced", func(t *testing.T) {
		desired := []VolumeMount{
			{
				Name:      "test-vol",
				MountPath: "/data",
				SubPath:   "subdir",
				ReadOnly:  true,
				MountOptions: &VolumeMountOptions{
					SELinuxLabel: "z",
					UIDMapping: &UIDGIDMapping{
						ContainerID: 1000,
						HostID:      100000,
						Size:        1,
					},
				},
			},
		}

		actual := []VolumeMount{
			{
				Name:      "test-vol",
				MountPath: "/data",
				SubPath:   "subdir",
				ReadOnly:  true,
				MountOptions: &VolumeMountOptions{
					SELinuxLabel: "z",
					UIDMapping: &UIDGIDMapping{
						ContainerID: 1000,
						HostID:      100000,
						Size:        1,
					},
				},
			},
		}

		if !cm.compareVolumes(desired, actual) {
			t.Error("Expected volumes to be equal")
		}

		// Test with different subPath
		actual[0].SubPath = "different-subdir"
		if cm.compareVolumes(desired, actual) {
			t.Error("Expected volumes to be different due to subPath")
		}

		// Test with different SELinux label
		actual[0].SubPath = "subdir"
		actual[0].MountOptions.SELinuxLabel = "Z"
		if cm.compareVolumes(desired, actual) {
			t.Error("Expected volumes to be different due to SELinux label")
		}
	})
}

// Helper function to check if a slice contains a string
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
