package resource

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewVolumePathManager(t *testing.T) {
	tests := []struct {
		name        string
		tempDirBase string
		expected    string
	}{
		{
			name:        "default temp dir",
			tempDirBase: "",
			expected:    "/tmp/cutepod-volumes",
		},
		{
			name:        "custom temp dir",
			tempDirBase: "/custom/temp",
			expected:    "/custom/temp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vpm := NewVolumePathManager(tt.tempDirBase)
			if vpm.tempDirBase != tt.expected {
				t.Errorf("expected tempDirBase %s, got %s", tt.expected, vpm.tempDirBase)
			}
		})
	}
}

func TestVolumePathManager_validateSubPath(t *testing.T) {
	vpm := NewVolumePathManager("")

	tests := []struct {
		name    string
		subPath string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty subPath is valid",
			subPath: "",
			wantErr: false,
		},
		{
			name:    "simple relative path",
			subPath: "config/app.conf",
			wantErr: false,
		},
		{
			name:    "path traversal with ..",
			subPath: "../etc/passwd",
			wantErr: true,
			errMsg:  "subPath cannot contain '..'",
		},
		{
			name:    "absolute path",
			subPath: "/etc/passwd",
			wantErr: true,
			errMsg:  "subPath must be relative",
		},
		{
			name:    "consecutive slashes",
			subPath: "config//app.conf",
			wantErr: true,
			errMsg:  "consecutive slashes",
		},
		{
			name:    "empty path component",
			subPath: "config/",
			wantErr: true,
			errMsg:  "empty path components",
		},
		{
			name:    "null character",
			subPath: "config\x00app.conf",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "newline character",
			subPath: "config\napp.conf",
			wantErr: true,
			errMsg:  "invalid character",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vpm.validateSubPath(tt.subPath)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func TestVolumePathManager_ResolveVolumePath_HostPath(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()
	vpm := NewVolumePathManager("")

	// Create test directory structure
	testDir := filepath.Join(tempDir, "test")
	os.MkdirAll(testDir, 0755)

	// Create test file
	testFile := filepath.Join(testDir, "config.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	tests := []struct {
		name           string
		volume         *VolumeResource
		mount          *VolumeMount
		expectedPath   string
		expectedIsFile bool
		expectedCreate bool
		wantErr        bool
		errMsg         string
	}{
		{
			name: "hostPath directory without subPath",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: testDir,
					},
				},
			},
			mount: &VolumeMount{
				Name:      "test-vol",
				MountPath: "/app/data",
			},
			expectedPath:   testDir,
			expectedIsFile: false,
			expectedCreate: false,
			wantErr:        false,
		},
		{
			name: "hostPath directory with subPath",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: testDir,
					},
				},
			},
			mount: &VolumeMount{
				Name:      "test-vol",
				MountPath: "/app/data",
				SubPath:   "subdir",
			},
			expectedPath:   filepath.Join(testDir, "subdir"),
			expectedIsFile: false,
			expectedCreate: true,
			wantErr:        false,
		},
		{
			name: "hostPath file with subPath",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: testDir,
					},
				},
			},
			mount: &VolumeMount{
				Name:      "test-vol",
				MountPath: "/app/config.txt",
				SubPath:   "config.txt",
			},
			expectedPath:   testFile,
			expectedIsFile: true,
			expectedCreate: false,
			wantErr:        false,
		},
		{
			name: "invalid subPath with path traversal",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: testDir,
					},
				},
			},
			mount: &VolumeMount{
				Name:      "test-vol",
				MountPath: "/app/data",
				SubPath:   "../../../etc/passwd",
			},
			wantErr: true,
			errMsg:  "subPath cannot contain '..'",
		},
		{
			name: "missing hostPath spec",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
				},
			},
			mount: &VolumeMount{
				Name:      "test-vol",
				MountPath: "/app/data",
			},
			wantErr: true,
			errMsg:  "hostPath specification is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathInfo, err := vpm.ResolveVolumePath(tt.volume, tt.mount)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error but got: %s", err.Error())
				return
			}

			if pathInfo.SourcePath != tt.expectedPath {
				t.Errorf("expected SourcePath %s, got %s", tt.expectedPath, pathInfo.SourcePath)
			}
			if pathInfo.IsFile != tt.expectedIsFile {
				t.Errorf("expected IsFile %v, got %v", tt.expectedIsFile, pathInfo.IsFile)
			}
			if pathInfo.RequiresCreation != tt.expectedCreate {
				t.Errorf("expected RequiresCreation %v, got %v", tt.expectedCreate, pathInfo.RequiresCreation)
			}
		})
	}
}

func TestVolumePathManager_ResolveVolumePath_EmptyDir(t *testing.T) {
	vpm := NewVolumePathManager("/tmp/test-cutepod")

	tests := []struct {
		name         string
		volume       *VolumeResource
		mount        *VolumeMount
		expectedPath string
		wantErr      bool
		errMsg       string
	}{
		{
			name: "emptyDir without subPath",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "temp-vol"},
				},
				Spec: CuteVolumeSpec{
					Type:     VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{},
				},
			},
			mount: &VolumeMount{
				Name:      "temp-vol",
				MountPath: "/tmp/data",
			},
			expectedPath: "/tmp/test-cutepod/emptydir/temp-vol",
			wantErr:      false,
		},
		{
			name: "emptyDir with subPath",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "temp-vol"},
				},
				Spec: CuteVolumeSpec{
					Type:     VolumeTypeEmptyDir,
					EmptyDir: &EmptyDirVolumeSource{},
				},
			},
			mount: &VolumeMount{
				Name:      "temp-vol",
				MountPath: "/tmp/data",
				SubPath:   "cache",
			},
			expectedPath: "/tmp/test-cutepod/emptydir/temp-vol/cache",
			wantErr:      false,
		},
		{
			name: "missing emptyDir spec",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "temp-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeEmptyDir,
				},
			},
			mount: &VolumeMount{
				Name:      "temp-vol",
				MountPath: "/tmp/data",
			},
			wantErr: true,
			errMsg:  "emptyDir specification is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathInfo, err := vpm.ResolveVolumePath(tt.volume, tt.mount)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error but got: %s", err.Error())
				return
			}

			if pathInfo.SourcePath != tt.expectedPath {
				t.Errorf("expected SourcePath %s, got %s", tt.expectedPath, pathInfo.SourcePath)
			}
			if pathInfo.IsFile != false {
				t.Errorf("expected IsFile false for emptyDir, got %v", pathInfo.IsFile)
			}
			if pathInfo.RequiresCreation != true {
				t.Errorf("expected RequiresCreation true for emptyDir, got %v", pathInfo.RequiresCreation)
			}
		})
	}
}

func TestVolumePathManager_EnsureVolumePath(t *testing.T) {
	tempDir := t.TempDir()
	vpm := NewVolumePathManager("")

	tests := []struct {
		name     string
		pathInfo *VolumePathInfo
		volume   *VolumeResource
		wantErr  bool
		validate func(t *testing.T, path string)
	}{
		{
			name: "create directory",
			pathInfo: &VolumePathInfo{
				SourcePath:       filepath.Join(tempDir, "new-dir"),
				IsFile:           false,
				RequiresCreation: true,
				PathType:         HostPathDirectoryOrCreate,
			},
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: tempDir,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, path string) {
				if stat, err := os.Stat(path); err != nil {
					t.Errorf("directory was not created: %v", err)
				} else if !stat.IsDir() {
					t.Errorf("expected directory, got file")
				}
			},
		},
		{
			name: "create file",
			pathInfo: &VolumePathInfo{
				SourcePath:       filepath.Join(tempDir, "new-file.txt"),
				IsFile:           true,
				RequiresCreation: true,
				PathType:         HostPathFileOrCreate,
			},
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: tempDir,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, path string) {
				if stat, err := os.Stat(path); err != nil {
					t.Errorf("file was not created: %v", err)
				} else if stat.IsDir() {
					t.Errorf("expected file, got directory")
				}
			},
		},
		{
			name: "no creation required",
			pathInfo: &VolumePathInfo{
				SourcePath:       tempDir,
				IsFile:           false,
				RequiresCreation: false,
				PathType:         HostPathDirectory,
			},
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: tempDir,
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, path string) {
				// Should not modify existing directory
			},
		},
		{
			name: "create directory with ownership",
			pathInfo: &VolumePathInfo{
				SourcePath:       filepath.Join(tempDir, "owned-dir"),
				IsFile:           false,
				RequiresCreation: true,
				PathType:         HostPathDirectoryOrCreate,
			},
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{
					Type: VolumeTypeHostPath,
					HostPath: &HostPathVolumeSource{
						Path: tempDir,
					},
					SecurityContext: &VolumeSecurityContext{
						Owner: &VolumeOwnership{
							User:  &[]int64{1000}[0],
							Group: &[]int64{1000}[0],
						},
					},
				},
			},
			wantErr: false,
			validate: func(t *testing.T, path string) {
				if stat, err := os.Stat(path); err != nil {
					t.Errorf("directory was not created: %v", err)
				} else if !stat.IsDir() {
					t.Errorf("expected directory, got file")
				}
				// Note: ownership validation would require running as root or checking syscalls
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vpm.EnsureVolumePath(tt.pathInfo, tt.volume)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("expected no error but got: %s", err.Error())
				return
			}

			if tt.validate != nil {
				tt.validate(t, tt.pathInfo.SourcePath)
			}
		})
	}
}

func TestVolumePathManager_CleanupEmptyDirVolume(t *testing.T) {
	tempDir := t.TempDir()
	vpm := NewVolumePathManager(tempDir)

	// Create a test emptyDir volume directory
	volumeName := "test-empty-vol"
	emptyDirPath := filepath.Join(tempDir, "emptydir", volumeName)
	os.MkdirAll(emptyDirPath, 0755)

	// Create some test files in it
	testFile := filepath.Join(emptyDirPath, "test.txt")
	os.WriteFile(testFile, []byte("test"), 0644)

	// Verify it exists
	if _, err := os.Stat(emptyDirPath); err != nil {
		t.Fatalf("test setup failed: %v", err)
	}

	// Clean it up
	err := vpm.CleanupEmptyDirVolume(volumeName)
	if err != nil {
		t.Errorf("cleanup failed: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(emptyDirPath); !os.IsNotExist(err) {
		t.Errorf("emptyDir volume was not cleaned up")
	}
}

func TestVolumePathManager_shouldCreateAsFile(t *testing.T) {
	vpm := NewVolumePathManager("")

	tests := []struct {
		name     string
		pathType HostPathType
		subPath  string
		expected bool
	}{
		{
			name:     "File type",
			pathType: HostPathFile,
			subPath:  "",
			expected: true,
		},
		{
			name:     "FileOrCreate type",
			pathType: HostPathFileOrCreate,
			subPath:  "",
			expected: true,
		},
		{
			name:     "Directory type",
			pathType: HostPathDirectory,
			subPath:  "",
			expected: false,
		},
		{
			name:     "DirectoryOrCreate type",
			pathType: HostPathDirectoryOrCreate,
			subPath:  "",
			expected: false,
		},
		{
			name:     "infer from subPath extension - file",
			pathType: HostPathDirectoryOrCreate,
			subPath:  "config.txt",
			expected: true,
		},
		{
			name:     "infer from subPath extension - no extension",
			pathType: HostPathDirectoryOrCreate,
			subPath:  "config",
			expected: false,
		},
		{
			name:     "infer from subPath - directory path",
			pathType: HostPathDirectoryOrCreate,
			subPath:  "configs/app",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := vpm.shouldCreateAsFile(tt.pathType, tt.subPath)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHostPathValidator_validateHostPath(t *testing.T) {
	tests := []struct {
		name            string
		allowedPrefixes []string
		hostPath        string
		wantErr         bool
		errMsg          string
	}{
		{
			name:     "valid absolute path",
			hostPath: "/home/user/data",
			wantErr:  false,
		},
		{
			name:     "relative path",
			hostPath: "relative/path",
			wantErr:  true,
			errMsg:   "hostPath must be an absolute path",
		},
		{
			name:     "path with traversal",
			hostPath: "/home/../etc/passwd",
			wantErr:  true,
			errMsg:   "hostPath cannot contain '..'",
		},
		{
			name:     "path with invalid components",
			hostPath: "/home/./user",
			wantErr:  true,
			errMsg:   "invalid path components",
		},
		{
			name:            "allowed prefix - valid",
			allowedPrefixes: []string{"/home", "/tmp"},
			hostPath:        "/home/user/data",
			wantErr:         false,
		},
		{
			name:            "allowed prefix - invalid",
			allowedPrefixes: []string{"/home", "/tmp"},
			hostPath:        "/etc/passwd",
			wantErr:         true,
			errMsg:          "not within allowed prefixes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := &HostPathValidator{
				allowedPrefixes: tt.allowedPrefixes,
			}

			err := validator.validateHostPath(tt.hostPath)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Error())
				}
			}
		})
	}
}

func TestVolumePathManager_ResolveVolumePath_NilInputs(t *testing.T) {
	vpm := NewVolumePathManager("")

	tests := []struct {
		name    string
		volume  *VolumeResource
		mount   *VolumeMount
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil volume",
			volume:  nil,
			mount:   &VolumeMount{Name: "test"},
			wantErr: true,
			errMsg:  "volume resource cannot be nil",
		},
		{
			name: "nil mount",
			volume: &VolumeResource{
				BaseResource: BaseResource{
					ObjectMeta: metav1.ObjectMeta{Name: "test-vol"},
				},
				Spec: CuteVolumeSpec{Type: VolumeTypeHostPath},
			},
			mount:   nil,
			wantErr: true,
			errMsg:  "volume mount cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := vpm.ResolveVolumePath(tt.volume, tt.mount)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %s", err.Error())
				}
			}
		})
	}
}
