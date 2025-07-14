package container

import (
	"cutepod/internal/object"
	"testing"

	types "github.com/containers/podman/v5/libpod/define"
	in "github.com/containers/podman/v5/pkg/inspect"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCompare_NoChanges(t *testing.T) {
	uid := int64(1000)

	spec := CuteContainerSpec{
		Image:   "alpine:latest",
		Command: []string{"/bin/sh"},
		Args:    []string{"-c", "echo hello"},
		UID:     &uid,
		Ports: []ContainerPort{
			{ContainerPort: 80, HostPort: 8080},
		},
	}

	inspect := &types.InspectContainerData{
		Name: "test-container",
		Config: &types.InspectContainerConfig{
			Image: "alpine:latest",
			Cmd:   []string{"/bin/sh"},
			User:  "1000",
		},
		Args: []string{"-c", "echo hello"},
		HostConfig: &types.InspectContainerHostConfig{
			PortBindings: map[string][]types.InspectHostPort{
				"80/tcp": {
					{HostIP: "0.0.0.0", HostPort: "8080"},
				},
			},
		},
	}

	container := &CuteContainer{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name: "container",
		},
		Spec: spec,
	}

	target := object.NewInstallTarget("test")

	changes, err := Compare(*target, container, inspect, data())
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes, got: %+v", changes)
	}
}

func data() *in.ImageData {
	return &in.ImageData{
		Config: &v1.ImageConfig{},
	}
}

func TestCompare_HandleMissingUID(t *testing.T) {
	spec := CuteContainerSpec{
		Image:   "alpine",
		Command: []string{"sh"},
		Args:    []string{"-c", "echo hi"},
		UID:     nil, // UID not set
	}

	inspect := &types.InspectContainerData{
		Name: "default-ct",
		Config: &types.InspectContainerConfig{
			Image: "alpine",
			Cmd:   []string{"sh"},
			User:  "", // interpreted as UID unset
		},
		Args:       []string{"-c", "echo hi"},
		HostConfig: &types.InspectContainerHostConfig{},
	}

	container := &CuteContainer{Spec: spec, ObjectMeta: metav1.ObjectMeta{Name: "ct"}}
	target := *object.NewInstallTarget("")

	changes, err := Compare(target, container, inspect, data())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("expected no changes, got: %+v", changes)
	}
}

func TestCompare_ImageDefaultsToLatest(t *testing.T) {
	spec := CuteContainerSpec{
		Image:   "alpine", // no tag specified
		Command: []string{"sh"},
		Args:    []string{"-c", "echo ok"},
	}

	inspect := &types.InspectContainerData{
		Name: "ct",
		Config: &types.InspectContainerConfig{
			Image: "alpine:latest", // what Podman returns
			Cmd:   []string{"sh"},
		},
		Args:       []string{"-c", "echo ok"},
		HostConfig: &types.InspectContainerHostConfig{},
	}

	c := &CuteContainer{Spec: spec, ObjectMeta: metav1.ObjectMeta{Name: "ct"}}
	target := *object.NewInstallTarget("")

	changes, err := Compare(target, c, inspect, data())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, change := range changes {
		if change.Path == "spec.image" {
			t.Errorf("expected no diff in image version, got: %+v", change)
		}
	}
}
func TestCompare_NormalizeArgsFallback(t *testing.T) {
	image := &in.ImageData{
		Config: &v1.ImageConfig{
			Entrypoint: []string{"/start.sh", "--flag"},
		},
	}

	inspect := &types.InspectContainerData{
		Name: "ct",
		Config: &types.InspectContainerConfig{
			Image: "myimage:latest",
			Cmd:   []string{"/start.sh", "--flag"}, // matches image.Entrypoint
		},
		HostConfig: &types.InspectContainerHostConfig{},
		Args:       nil, // simulate podman inspect output
	}

	c := &CuteContainer{
		Spec: CuteContainerSpec{
			Image:   "myimage", // should normalize to myimage:latest
			Command: []string{"/start.sh"},
			Args:    nil, // This will trigger normalizeArgs fallback
		},
	}

	target := *object.NewInstallTarget("test")

	changes, err := Compare(target, c, inspect, image)
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}

	for _, change := range changes {
		if change.Path == "spec.args" {
			t.Errorf("Expected args to match via image.Entrypoint fallback, got unexpected change: %+v", change)
		}
	}
}

func TestCompare_NormalizeWorkindDir(t *testing.T) {
	image := &in.ImageData{
		Config: &v1.ImageConfig{
			WorkingDir: "foo/bar",
		},
	}

	inspect := &types.InspectContainerData{
		Name: "ct",
		Config: &types.InspectContainerConfig{
			Image:      "myimage:latest",
			WorkingDir: "foo/bar",
		},
		HostConfig: &types.InspectContainerHostConfig{},
		Args:       nil, // simulate podman inspect output
	}

	c := &CuteContainer{
		Spec: CuteContainerSpec{
			Image:      "myimage", // should normalize to myimage:latest
			WorkingDir: "",        // This will trigger normalizeArgs fallback
		},
	}

	target := *object.NewInstallTarget("test")

	changes, err := Compare(target, c, inspect, image)
	if err != nil {
		t.Fatalf("Compare returned error: %v", err)
	}

	for _, change := range changes {
		if change.Path == "spec.args" {
			t.Errorf("Expected args to match via image.Entrypoint fallback, got unexpected change: %+v", change)
		}
	}
}
