package resource

import (
	"strings"
	"testing"
)

func TestContainerResource_ValidateVolumeMounts(t *testing.T) {
	tests := []struct {
		name        string
		spec        CuteContainerSpec
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid volume mount",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						SubPath:   "subdir",
						ReadOnly:  false,
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      subPath: subdir
`,
			expectError: false,
		},
		{
			name: "empty volume name",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "",
						MountPath: "/data",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: ""
      mountPath: /data
`,
			expectError: true,
			errorMsg:    "volume name must not be empty",
		},
		{
			name: "empty mount path",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: ""
`,
			expectError: true,
			errorMsg:    "mountPath must not be empty",
		},
		{
			name: "relative mount path",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "data",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: data
`,
			expectError: true,
			errorMsg:    "mountPath must be an absolute path starting with '/'",
		},
		{
			name: "subPath with path traversal",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						SubPath:   "../etc/passwd",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      subPath: ../etc/passwd
`,
			expectError: true,
			errorMsg:    "subPath must not contain '..' (path traversal not allowed)",
		},
		{
			name: "subPath with absolute path",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						SubPath:   "/absolute/path",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      subPath: /absolute/path
`,
			expectError: true,
			errorMsg:    "subPath must be a relative path (cannot start with '/')",
		},
		{
			name: "subPath with consecutive slashes",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						SubPath:   "path//with//double//slashes",
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      subPath: path//with//double//slashes
`,
			expectError: true,
			errorMsg:    "subPath must not contain consecutive slashes",
		},
		{
			name: "invalid SELinux label",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						MountOptions: &VolumeMountOptions{
							SELinuxLabel: "invalid",
						},
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      mountOptions:
        seLinuxLabel: invalid
`,
			expectError: true,
			errorMsg:    "seLinuxLabel must be one of: z, Z, shared, private",
		},
		{
			name: "valid SELinux labels",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data1",
						MountPath: "/data1",
						MountOptions: &VolumeMountOptions{
							SELinuxLabel: "z",
						},
					},
					{
						Name:      "data2",
						MountPath: "/data2",
						MountOptions: &VolumeMountOptions{
							SELinuxLabel: "Z",
						},
					},
					{
						Name:      "data3",
						MountPath: "/data3",
						MountOptions: &VolumeMountOptions{
							SELinuxLabel: "shared",
						},
					},
					{
						Name:      "data4",
						MountPath: "/data4",
						MountOptions: &VolumeMountOptions{
							SELinuxLabel: "private",
						},
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data1
      mountPath: /data1
      mountOptions:
        seLinuxLabel: z
    - name: data2
      mountPath: /data2
      mountOptions:
        seLinuxLabel: Z
    - name: data3
      mountPath: /data3
      mountOptions:
        seLinuxLabel: shared
    - name: data4
      mountPath: /data4
      mountOptions:
        seLinuxLabel: private
`,
			expectError: false,
		},
		{
			name: "invalid UID mapping size",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						MountOptions: &VolumeMountOptions{
							UIDMapping: &UIDGIDMapping{
								ContainerID: 1000,
								HostID:      100000,
								Size:        0, // Invalid size
							},
						},
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      mountOptions:
        uidMapping:
          containerID: 1000
          hostID: 100000
          size: 0
`,
			expectError: true,
			errorMsg:    "uidMapping.size must be greater than 0",
		},
		{
			name: "invalid GID mapping size",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						MountOptions: &VolumeMountOptions{
							GIDMapping: &UIDGIDMapping{
								ContainerID: 1000,
								HostID:      100000,
								Size:        -1, // Invalid size
							},
						},
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      mountOptions:
        gidMapping:
          containerID: 1000
          hostID: 100000
          size: -1
`,
			expectError: true,
			errorMsg:    "gidMapping.size must be greater than 0",
		},
		{
			name: "valid UID/GID mapping",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:      "data",
						MountPath: "/data",
						MountOptions: &VolumeMountOptions{
							UIDMapping: &UIDGIDMapping{
								ContainerID: 1000,
								HostID:      100000,
								Size:        1,
							},
							GIDMapping: &UIDGIDMapping{
								ContainerID: 1000,
								HostID:      100000,
								Size:        1,
							},
						},
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      mountPath: /data
      mountOptions:
        uidMapping:
          containerID: 1000
          hostID: 100000
          size: 1
        gidMapping:
          containerID: 1000
          hostID: 100000
          size: 1
`,
			expectError: false,
		},
		{
			name: "backward compatibility with containerPath",
			spec: CuteContainerSpec{
				Image: "nginx:latest",
				Volumes: []VolumeMount{
					{
						Name:          "data",
						ContainerPath: "/data", // Deprecated field
						MountPath:     "",      // Empty new field
					},
				},
			},
			yaml: `
spec:
  image: nginx:latest
  volumes:
    - name: data
      containerPath: /data
`,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := &ContainerResource{
				BaseResource: BaseResource{
					ResourceType: ResourceTypeContainer,
				},
				Spec: tt.spec,
			}

			errs := container.Validate(tt.yaml)

			if tt.expectError {
				if len(errs) == 0 {
					t.Errorf("expected validation error but got none")
					return
				}

				found := false
				for _, err := range errs {
					if strings.Contains(err.Error(), tt.errorMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected error message containing '%s', but got: %v", tt.errorMsg, errs)
				}
			} else {
				if len(errs) > 0 {
					t.Errorf("expected no validation errors but got: %v", errs)
				}
			}
		})
	}
}
