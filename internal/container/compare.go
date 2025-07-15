package container

import (
	"cutepod/internal/object"
	"fmt"
	"strconv"
	"strings"

	"github.com/containers/podman/v5/libpod/define"
	"github.com/containers/podman/v5/pkg/inspect"
)

func Compare(t object.InstallTarget, c *CuteContainer, inspect *define.InspectContainerData, image *inspect.ImageData) ([]object.SpecChange, error) {
	var changes []object.SpecChange
	changes = stringChanges(changes, "metadata.name", t.GetContainerName(c), inspect.Name)
	changes = stringChanges(changes, "spec.image", normalizeImage(c.Spec.Image), normalizeImage(inspect.Config.Image))
	changes = stringArrayChanges(changes, "spec.command", c.Spec.Command, inspect.Config.Cmd)
	changes = stringArrayChanges(changes, "spec.args", normalizeArgs(c.Spec.Args, image), inspect.Args)

	// Working directory
	changes = stringChanges(changes, "spec.workingDir", normalizeWorkingDir(c.Spec.WorkingDir, image.Config.WorkingDir), inspect.Config.WorkingDir)

	// UID & GID
	if c.Spec.UID != nil && inspect.Config.User != "" {
		if uidStr := strconv.FormatInt(*c.Spec.UID, 10); uidStr != inspect.Config.User {
			changes = append(changes, object.SpecChange{
				Path:     "spec.uid",
				Expected: uidStr,
				Actual:   inspect.Config.User,
			})
		}
	}

	// Env vars
	changes = compareEnvVars(changes, "spec.env", c.Spec.Env, inspect.Config.Env)

	// Ports
	changes = comparePorts(changes, "spec.ports", c.Spec.Ports, inspect.HostConfig.PortBindings)

	// Volumes
	changes = compareVolumes(changes, "spec.volumes", c.Spec.Volumes, inspect.Mounts)

	// RestartPolicy
	if inspect.HostConfig.RestartPolicy != nil {
		changes = stringChanges(changes, "spec.restartPolicy", normlizePolicy(c.Spec.RestartPolicy), inspect.HostConfig.RestartPolicy.Name)
	}

	// Capabilities
	if inspect.HostConfig != nil && c.Spec.SecurityContext != nil {
		if c.Spec.SecurityContext.Capabilities != nil {
			changes = stringArrayChanges(changes, "spec.securityContext.capabilities.add", c.Spec.SecurityContext.Capabilities.Add, inspect.HostConfig.CapAdd)
			changes = stringArrayChanges(changes, "spec.securityContext.capabilities.drop", c.Spec.SecurityContext.Capabilities.Drop, inspect.HostConfig.CapDrop)
		}
		if c.Spec.SecurityContext.Privileged != nil {
			changes = maybeChangeBool(changes, "spec.securityContext.privileged", *c.Spec.SecurityContext.Privileged, inspect.HostConfig.Privileged)
		}
	}

	// Resource limits
	if c.Spec.Resources != nil {
		if c.Spec.Resources.Limits.Memory != "" {
			changes = stringChanges(changes, "spec.resources.limits.memory", c.Spec.Resources.Limits.Memory, fmt.Sprintf("%d", inspect.HostConfig.Memory))
		}
		if c.Spec.Resources.Limits.CPU != "" {
			changes = stringChanges(changes, "spec.resources.limits.cpu", c.Spec.Resources.Limits.CPU, fmt.Sprintf("%d", inspect.HostConfig.NanoCpus))
		}
	}

	return changes, nil
}

func normlizePolicy(policy string) string {
	if policy == "" {
		return "no"
	}
	return policy
}

func normalizeWorkingDir(dir string, dir2 string) string {
	if dir == "" {
		return dir2
	}
	return dir
}

func normalizeArgs(args []string, image *inspect.ImageData) []string {
	if args == nil || len(args) == 0 {
		return image.Config.Entrypoint
	}

	return args
}

func normalizeImage(image string) string {
	if !strings.Contains(image, ":") {
		return image + ":latest"
	}
	return image
}

func compareEnvVars(changes []object.SpecChange, path string, expected []EnvVar, actual []string) []object.SpecChange {
	envMap := make(map[string]string)
	for _, env := range expected {
		envMap[env.Name] = env.Value
	}
	for _, a := range actual {
		parts := strings.SplitN(a, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := parts[0], parts[1]
		if ev, ok := envMap[k]; ok && ev != v {
			changes = append(changes, object.SpecChange{
				Path:     path + "." + k,
				Actual:   v,
				Expected: ev,
			})
		}
		delete(envMap, k)
	}
	for k, v := range envMap {
		changes = append(changes, object.SpecChange{
			Path:     path + "." + k,
			Actual:   "<missing>",
			Expected: v,
		})
	}
	return changes
}

func comparePorts(changes []object.SpecChange, path string, expected []ContainerPort, actual map[string][]define.InspectHostPort) []object.SpecChange {
	for _, port := range expected {
		protocol := port.Protocol
		if protocol == "" {
			protocol = "tcp"
		}
		key := fmt.Sprintf("%d/%s", port.ContainerPort, strings.ToLower(protocol))
		actualBindings, ok := actual[key]
		if !ok || len(actualBindings) == 0 {
			changes = append(changes, object.SpecChange{
				Path:     path + "." + key,
				Actual:   "<missing>",
				Expected: fmt.Sprintf("%d->%d", port.HostPort, port.ContainerPort),
			})
			continue
		}
		found := false
		for _, binding := range actualBindings {
			hostPort, _ := strconv.Atoi(binding.HostPort)
			if uint16(hostPort) == port.HostPort {
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, object.SpecChange{
				Path:     path + "." + key,
				Actual:   fmt.Sprintf("%v", actualBindings),
				Expected: fmt.Sprintf("%d", port.HostPort),
			})
		}
	}
	return changes
}
func compareVolumes(changes []object.SpecChange, path string, expected []VolumeMount, actual []define.InspectMount) []object.SpecChange {
	for _, vol := range expected {
		found := false
		for _, m := range actual {
			if m.Destination == vol.ContainerPath {
				if m.Source != vol.HostPath || m.RW == vol.ReadOnly {
					changes = append(changes, object.SpecChange{
						Path:     path + "." + vol.ContainerPath,
						Actual:   fmt.Sprintf("source=%s, readonly=%t", m.Source, !m.RW),
						Expected: fmt.Sprintf("source=%s, readonly=%t", vol.HostPath, vol.ReadOnly),
					})
				}
				found = true
				break
			}
		}
		if !found {
			changes = append(changes, object.SpecChange{
				Path:     path + "." + vol.ContainerPath,
				Actual:   "<missing>",
				Expected: fmt.Sprintf("source=%s, readonly=%t", vol.HostPath, vol.ReadOnly),
			})
		}
	}
	return changes
}
func maybeChangeBool(changes []object.SpecChange, path string, expected bool, actual bool) []object.SpecChange {
	if expected != actual {
		return append(changes, object.SpecChange{
			Path:     path,
			Actual:   fmt.Sprintf("%v", actual),
			Expected: fmt.Sprintf("%v", expected),
		})
	}
	return changes
}

func stringArrayChanges(changes []object.SpecChange, path string, expected []string, actual []string) []object.SpecChange {
	if len(expected) != len(actual) {
		return append(changes, object.SpecChange{
			Path:     path + ".size",
			Actual:   strconv.Itoa(len(actual)),
			Expected: strconv.Itoa(len(expected)),
		})
	}

	for i, e := range expected {
		if e != actual[i] {
			changes = append(changes, object.SpecChange{
				Path:     path + ".[" + strconv.Itoa(i) + "]",
				Actual:   actual[i],
				Expected: e,
			})
		}
	}

	return changes
}
func stringChanges(changes []object.SpecChange, path string, expected string, actual string) []object.SpecChange {
	if actual != expected {
		return append(changes, object.SpecChange{
			Path:     path,
			Actual:   actual,
			Expected: expected,
		})
	}

	return changes
}
