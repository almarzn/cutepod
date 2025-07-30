package resource

import (
	"fmt"
)

// StateComparator handles the core logic of comparing desired vs actual state
type StateComparator interface {
	// CompareStates compares desired vs actual resources and returns a diff
	CompareStates(desired, actual []Resource) (*StateDiff, error)

	// ShouldUpdate determines if a resource should be updated and returns the reasons
	ShouldUpdate(desired, actual Resource) (bool, []string, error)

	// SetResourceManager sets the resource manager for a specific resource type
	SetResourceManager(resourceType ResourceType, manager ResourceManager)
}

// StateDiff represents the differences between desired and actual state
type StateDiff struct {
	ToCreate  []Resource     `json:"to_create"`
	ToUpdate  []ResourcePair `json:"to_update"`
	ToDelete  []Resource     `json:"to_delete"`
	Unchanged []Resource     `json:"unchanged"`
}

// ResourcePair represents a pair of desired and actual resources for comparison
type ResourcePair struct {
	Desired Resource `json:"desired"`
	Actual  Resource `json:"actual"`
}

// DefaultStateComparator implements StateComparator
type DefaultStateComparator struct {
	managers map[ResourceType]ResourceManager
}

// NewStateComparator creates a new state comparator
func NewStateComparator() StateComparator {
	return &DefaultStateComparator{
		managers: make(map[ResourceType]ResourceManager),
	}
}

// SetResourceManager sets the resource manager for a specific resource type
func (sc *DefaultStateComparator) SetResourceManager(resourceType ResourceType, manager ResourceManager) {
	sc.managers[resourceType] = manager
}

// CompareStates compares desired vs actual resources and returns a diff
func (sc *DefaultStateComparator) CompareStates(desired, actual []Resource) (*StateDiff, error) {
	diff := &StateDiff{
		ToCreate:  make([]Resource, 0),
		ToUpdate:  make([]ResourcePair, 0),
		ToDelete:  make([]Resource, 0),
		Unchanged: make([]Resource, 0),
	}

	// Create maps for efficient lookup
	desiredMap := make(map[string]Resource)
	actualMap := make(map[string]Resource)

	for _, res := range desired {
		key := sc.getResourceKey(res)
		desiredMap[key] = res
	}

	for _, res := range actual {
		key := sc.getResourceKey(res)
		actualMap[key] = res
	}

	// Find resources to create and update
	for key, desiredRes := range desiredMap {
		if actualRes, exists := actualMap[key]; exists {
			// Resource exists, check if it needs updating
			shouldUpdate, _, err := sc.ShouldUpdate(desiredRes, actualRes)
			if err != nil {
				return nil, fmt.Errorf("failed to compare resources %s: %w", key, err)
			}

			if shouldUpdate {
				diff.ToUpdate = append(diff.ToUpdate, ResourcePair{
					Desired: desiredRes,
					Actual:  actualRes,
				})
			} else {
				diff.Unchanged = append(diff.Unchanged, desiredRes)
			}
		} else {
			// Resource doesn't exist, needs to be created
			diff.ToCreate = append(diff.ToCreate, desiredRes)
		}
	}

	// Find resources to delete (exist in actual but not in desired)
	for key, actualRes := range actualMap {
		if _, exists := desiredMap[key]; !exists {
			diff.ToDelete = append(diff.ToDelete, actualRes)
		}
	}

	return diff, nil
}

// ShouldUpdate determines if a resource should be updated
func (sc *DefaultStateComparator) ShouldUpdate(desired, actual Resource) (bool, []string, error) {
	if desired.GetType() != actual.GetType() {
		return false, nil, fmt.Errorf("resource type mismatch: desired=%s, actual=%s",
			desired.GetType(), actual.GetType())
	}

	if desired.GetName() != actual.GetName() {
		return false, nil, fmt.Errorf("resource name mismatch: desired=%s, actual=%s",
			desired.GetName(), actual.GetName())
	}

	// Use the appropriate resource manager for comparison
	manager, exists := sc.managers[desired.GetType()]
	if !exists {
		// Fallback to basic comparison if no manager is available
		return sc.basicComparison(desired, actual)
	}

	// Use the manager's comparison logic
	matches, err := manager.CompareResources(desired, actual)
	if err != nil {
		return false, nil, fmt.Errorf("failed to compare resources using manager: %w", err)
	}

	if matches {
		return false, []string{}, nil
	}

	// Resources don't match, determine the reasons
	reasons := sc.determineUpdateReasons(desired, actual)
	return true, reasons, nil
}

// basicComparison performs a basic comparison when no specific manager is available
func (sc *DefaultStateComparator) basicComparison(desired, actual Resource) (bool, []string, error) {
	reasons := make([]string, 0)

	// Compare labels
	desiredLabels := desired.GetLabels()
	actualLabels := actual.GetLabels()

	if !sc.compareMaps(desiredLabels, actualLabels) {
		reasons = append(reasons, "labels differ")
	}

	return len(reasons) > 0, reasons, nil
}

// determineUpdateReasons analyzes the differences between desired and actual resources
func (sc *DefaultStateComparator) determineUpdateReasons(desired, actual Resource) []string {
	reasons := make([]string, 0)

	desiredLabels := desired.GetLabels()
	actualLabels := actual.GetLabels()
	if !sc.compareMaps(desiredLabels, actualLabels) {
		reasons = append(reasons, "labels changed")
	}

	// Add resource-specific comparison logic
	switch desired.GetType() {
	case ResourceTypeContainer:
		reasons = append(reasons, sc.compareContainerResources(desired, actual)...)
	case ResourceTypeNetwork:
		reasons = append(reasons, sc.compareNetworkResources(desired, actual)...)
	case ResourceTypeVolume:
		reasons = append(reasons, sc.compareVolumeResources(desired, actual)...)
	case ResourceTypeSecret:
		reasons = append(reasons, sc.compareSecretResources(desired, actual)...)
	}

	if len(reasons) == 0 {
		reasons = append(reasons, "configuration changed")
	}

	return reasons
}

// compareContainerResources compares container-specific fields
func (sc *DefaultStateComparator) compareContainerResources(desired, actual Resource) []string {
	reasons := make([]string, 0)

	desiredContainer, ok1 := desired.(*ContainerResource)
	actualContainer, ok2 := actual.(*ContainerResource)

	if !ok1 || !ok2 {
		reasons = append(reasons, "resource type conversion failed")
		return reasons
	}

	if desiredContainer.Spec.Image != actualContainer.Spec.Image {
		reasons = append(reasons, "image changed")
	}

	if !sc.compareStringSlices(desiredContainer.Spec.Command, actualContainer.Spec.Command) {
		reasons = append(reasons, "command changed")
	}

	if !sc.compareStringSlices(desiredContainer.Spec.Args, actualContainer.Spec.Args) {
		reasons = append(reasons, "args changed")
	}

	if len(desiredContainer.Spec.Env) != len(actualContainer.Spec.Env) {
		reasons = append(reasons, "environment variables changed")
	}

	if len(desiredContainer.Spec.Ports) != len(actualContainer.Spec.Ports) {
		reasons = append(reasons, "ports changed")
	}

	if len(desiredContainer.Spec.Volumes) != len(actualContainer.Spec.Volumes) {
		reasons = append(reasons, "volumes changed")
	}

	return reasons
}

// compareNetworkResources compares network-specific fields
func (sc *DefaultStateComparator) compareNetworkResources(desired, actual Resource) []string {
	reasons := make([]string, 0)

	desiredNetwork, ok1 := desired.(*NetworkResource)
	actualNetwork, ok2 := actual.(*NetworkResource)

	if !ok1 || !ok2 {
		reasons = append(reasons, "resource type conversion failed")
		return reasons
	}

	if desiredNetwork.Spec.Driver != actualNetwork.Spec.Driver {
		reasons = append(reasons, "driver changed")
	}

	if desiredNetwork.Spec.Subnet != actualNetwork.Spec.Subnet {
		reasons = append(reasons, "subnet changed")
	}

	if !sc.compareMaps(desiredNetwork.Spec.Options, actualNetwork.Spec.Options) {
		reasons = append(reasons, "options changed")
	}

	return reasons
}

// compareVolumeResources compares volume-specific fields
func (sc *DefaultStateComparator) compareVolumeResources(desired, actual Resource) []string {
	reasons := make([]string, 0)

	desiredVolume, ok1 := desired.(*VolumeResource)
	actualVolume, ok2 := actual.(*VolumeResource)

	if !ok1 || !ok2 {
		reasons = append(reasons, "resource type conversion failed")
		return reasons
	}

	if desiredVolume.Spec.Type != actualVolume.Spec.Type {
		reasons = append(reasons, "volume type changed")
	}

	if desiredVolume.Spec.HostPath != actualVolume.Spec.HostPath {
		reasons = append(reasons, "host path changed")
	}

	if desiredVolume.Spec.EmptyDir != actualVolume.Spec.EmptyDir {
		reasons = append(reasons, "empty dir changed")
	}

	if desiredVolume.Spec.Volume != actualVolume.Spec.Volume {
		reasons = append(reasons, "volume spec changed")
	}

	if desiredVolume.Spec.SecurityContext != actualVolume.Spec.SecurityContext {
		reasons = append(reasons, "security context changed")
	}

	return reasons
}

// compareSecretResources compares secret-specific fields
func (sc *DefaultStateComparator) compareSecretResources(desired, actual Resource) []string {
	reasons := make([]string, 0)

	desiredSecret, ok1 := desired.(*SecretResource)
	actualSecret, ok2 := actual.(*SecretResource)

	if !ok1 || !ok2 {
		reasons = append(reasons, "resource type conversion failed")
		return reasons
	}

	if desiredSecret.Spec.Type != actualSecret.Spec.Type {
		reasons = append(reasons, "secret type changed")
	}

	if len(desiredSecret.Spec.Data) != len(actualSecret.Spec.Data) {
		reasons = append(reasons, "secret data changed")
	} else {
		// Compare secret data keys (not values for security)
		for key := range desiredSecret.Spec.Data {
			if _, exists := actualSecret.Spec.Data[key]; !exists {
				reasons = append(reasons, "secret data keys changed")
				break
			}
		}
	}

	return reasons
}

// Helper methods

func (sc *DefaultStateComparator) getResourceKey(resource Resource) string {
	return fmt.Sprintf("%s/%s", resource.GetType(), resource.GetName())
}

func (sc *DefaultStateComparator) compareMaps(map1, map2 map[string]string) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, value1 := range map1 {
		if value2, exists := map2[key]; !exists || value1 != value2 {
			return false
		}
	}

	return true
}

func (sc *DefaultStateComparator) compareStringSlices(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return false
	}

	for i, value1 := range slice1 {
		if value1 != slice2[i] {
			return false
		}
	}

	return true
}
