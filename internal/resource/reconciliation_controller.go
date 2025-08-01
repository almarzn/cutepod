package resource

import (
	"context"
	"cutepod/internal/podman"
	"fmt"
	"sync"
	"time"
)

// ReconciliationController orchestrates the complete reconciliation workflow
type ReconciliationController interface {
	// Reconcile performs the full reconciliation workflow: parse → resolve → compare → execute
	Reconcile(ctx context.Context, manifests []Resource, chartName string, dryRun bool) (*ReconciliationResult, error)

	// GetStatus returns the current reconciliation status for a chartName
	GetStatus(chartName string) (*ReconciliationStatus, error)
}

// ReconciliationResult contains the results of a reconciliation operation
type ReconciliationResult struct {
	CreatedResources []ResourceAction       `json:"created_resources"`
	UpdatedResources []ResourceAction       `json:"updated_resources"`
	DeletedResources []ResourceAction       `json:"deleted_resources"`
	Errors           []*ReconciliationError `json:"errors"`
	Summary          string                 `json:"summary"`
	Duration         time.Duration          `json:"duration"`
	ChartName        string                 `json:"chart_name"`
}

// ReconciliationStatus represents the current status of reconciliation for a chart name
type ReconciliationStatus struct {
	ChartName      string                 `json:"chart_name"`
	LastReconciled time.Time              `json:"last_reconciled"`
	ResourceCounts map[string]int         `json:"resource_counts"`
	Status         string                 `json:"status"`
	Errors         []*ReconciliationError `json:"errors,omitempty"`
}

// ResourceAction represents an action taken on a resource during reconciliation
type ResourceAction struct {
	Type      ResourceType  `json:"type"`
	Name      string        `json:"name"`
	Action    ActionType    `json:"action"`
	Message   string        `json:"message,omitempty"`
	Error     string        `json:"error,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// ActionType represents the type of action taken on a resource
type ActionType string

const (
	ActionCreate ActionType = "create"
	ActionUpdate ActionType = "update"
	ActionDelete ActionType = "delete"
	ActionSkip   ActionType = "skip"
)

// ErrorTypeComparison represents comparison-related errors
const ErrorTypeComparison ErrorType = "comparison"

// DefaultReconciliationController implements ReconciliationController
type DefaultReconciliationController struct {
	managers           map[ResourceType]ResourceManager
	stateComparator    StateComparator
	dependencyResolver DependencyResolver
	podmanClient       podman.PodmanClient
	mu                 sync.RWMutex // Protects concurrent access to status
	lastStatus         map[string]*ReconciliationStatus
}

// NewReconciliationController creates a new reconciliation controller
func NewReconciliationController(podmanClient podman.PodmanClient) ReconciliationController {
	return NewReconciliationControllerWithRegistry(podmanClient, nil)
}

// NewReconciliationControllerWithRegistry creates a new reconciliation controller with a registry
func NewReconciliationControllerWithRegistry(podmanClient podman.PodmanClient, registry *ManifestRegistry) ReconciliationController {
	controller := &DefaultReconciliationController{
		managers:           make(map[ResourceType]ResourceManager),
		stateComparator:    NewStateComparator(),
		dependencyResolver: NewDependencyResolver(),
		podmanClient:       podmanClient,
		lastStatus:         make(map[string]*ReconciliationStatus),
	}

	// Register resource managers
	if registry != nil {
		controller.managers[ResourceTypeContainer] = NewContainerManagerWithRegistry(podmanClient, registry)
	} else {
		controller.managers[ResourceTypeContainer] = NewContainerManager(podmanClient)
	}
	controller.managers[ResourceTypeNetwork] = NewNetworkManager(podmanClient)
	controller.managers[ResourceTypeVolume] = NewVolumeManager(podmanClient)
	controller.managers[ResourceTypeSecret] = NewSecretManager(podmanClient)

	// Set up state comparator with resource managers
	stateComparator := controller.stateComparator.(*DefaultStateComparator)
	for resourceType, manager := range controller.managers {
		stateComparator.SetResourceManager(resourceType, manager)
	}

	return controller
}

// NewReconciliationControllerWithURI creates a new reconciliation controller with a Podman URI
func NewReconciliationControllerWithURI(podmanURI string) ReconciliationController {
	adapter := podman.NewPodmanAdapter()
	return NewReconciliationController(adapter)
}

// NewReconciliationControllerWithURIAndRegistry creates a new reconciliation controller with a Podman URI and registry
func NewReconciliationControllerWithURIAndRegistry(podmanURI string, registry *ManifestRegistry) ReconciliationController {
	adapter := podman.NewPodmanAdapter()
	return NewReconciliationControllerWithRegistry(adapter, registry)
}

// Reconcile performs the complete reconciliation workflow: parse → resolve → compare → execute
func (rc *DefaultReconciliationController) Reconcile(ctx context.Context, manifests []Resource, chartName string, dryRun bool) (*ReconciliationResult, error) {
	startTime := time.Now()

	result := &ReconciliationResult{
		CreatedResources: make([]ResourceAction, 0),
		UpdatedResources: make([]ResourceAction, 0),
		DeletedResources: make([]ResourceAction, 0),
		Errors:           make([]*ReconciliationError, 0),
		ChartName:        chartName,
	}

	// Validate input parameters
	if len(manifests) == 0 {
		result.Duration = time.Since(startTime)
		result.Summary = "No resources to reconcile"
		return result, nil
	}

	// Step 1: Parse and validate manifests
	if err := rc.validateManifests(manifests); err != nil {
		return result, rc.addError(result, ErrorTypeValidation, ResourceReference{},
			fmt.Sprintf("manifest validation failed: %v", err), err, false)
	}

	// Step 2: Build dependency graph with error recovery
	dependencyGraph, err := rc.buildDependencyGraphWithRetry(ctx, manifests, result)
	if err != nil {
		return result, err
	}

	// Step 3: Get creation and deletion order
	creationOrder, err := rc.dependencyResolver.GetCreationOrder(dependencyGraph)
	if err != nil {
		return result, rc.addError(result, ErrorTypeDependency, ResourceReference{},
			fmt.Sprintf("failed to determine creation order: %v", err), err, false)
	}

	deletionOrder, err := rc.dependencyResolver.GetDeletionOrder(dependencyGraph)
	if err != nil {
		return result, rc.addError(result, ErrorTypeDependency, ResourceReference{},
			fmt.Sprintf("failed to determine deletion order: %v", err), err, false)
	}

	// Step 4: Get current state with error recovery
	actualStateByType, err := rc.getCurrentStateWithRetry(ctx, chartName, result)
	if err != nil {
		return result, err
	}

	// Step 5: Compare states and determine actions
	stateDiff, err := rc.compareAllStatesWithValidation(manifests, actualStateByType, result)
	if err != nil {
		return result, err
	}

	// Step 6: Execute changes with comprehensive error handling
	if dryRun {
		rc.populateDryRunResult(result, stateDiff)
	} else {
		rc.executeReconciliationWithRecovery(ctx, result, stateDiff, creationOrder, deletionOrder)
	}

	// Step 7: Clean up orphaned resources with error handling
	if !dryRun {
		rc.cleanupOrphanedResourcesWithRecovery(ctx, result, manifests, actualStateByType, deletionOrder)
	}

	// Step 8: Update status and generate summary
	rc.updateReconciliationStatus(chartName, result, startTime)
	result.Duration = time.Since(startTime)
	result.Summary = rc.generateSummary(result)

	return result, nil
}

// GetStatus returns the current reconciliation status for a chart name
func (rc *DefaultReconciliationController) GetStatus(chartName string) (*ReconciliationStatus, error) {
	rc.mu.RLock()
	cachedStatus, exists := rc.lastStatus[chartName]
	rc.mu.RUnlock()

	// If we have cached status, return it with current resource counts
	if exists {
		// Update with current resource counts
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		currentStatus := &ReconciliationStatus{
			ChartName:      chartName,
			LastReconciled: cachedStatus.LastReconciled,
			ResourceCounts: make(map[string]int),
			Status:         cachedStatus.Status,
			Errors:         cachedStatus.Errors,
		}

		// Get current resource counts for each type
		for resourceType, manager := range rc.managers {
			resources, err := manager.GetActualState(ctx, chartName)
			if err != nil {
				currentStatus.Errors = append(currentStatus.Errors, NewPodmanAPIError(
					ResourceReference{Type: resourceType},
					fmt.Sprintf("failed to get current status for %s: %v", resourceType, err),
					err,
					true,
				))
				continue
			}
			currentStatus.ResourceCounts[string(resourceType)] = len(resources)
		}

		// Update overall status based on current errors
		if len(currentStatus.Errors) == 0 {
			currentStatus.Status = "healthy"
		} else {
			currentStatus.Status = "degraded"
		}

		return currentStatus, nil
	}

	// No cached status, create a fresh one
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	status := &ReconciliationStatus{
		ChartName:      chartName,
		ResourceCounts: make(map[string]int),
		Status:         "unknown",
		LastReconciled: time.Time{}, // Zero time indicates never reconciled
	}

	// Get current resource counts for each type
	for resourceType, manager := range rc.managers {
		resources, err := manager.GetActualState(ctx, chartName)
		if err != nil {
			status.Errors = append(status.Errors, NewPodmanAPIError(
				ResourceReference{Type: resourceType},
				fmt.Sprintf("failed to get status for %s: %v", resourceType, err),
				err,
				true,
			))
			continue
		}
		status.ResourceCounts[string(resourceType)] = len(resources)
	}

	// Determine overall status
	if len(status.Errors) == 0 {
		status.Status = "healthy"
	} else {
		status.Status = "degraded"
	}

	return status, nil
}

// populateDryRunResult populates the result for dry run mode
func (rc *DefaultReconciliationController) populateDryRunResult(result *ReconciliationResult, diff *StateDiff) {
	now := time.Now()

	// Add create actions
	for _, resource := range diff.ToCreate {
		result.CreatedResources = append(result.CreatedResources, ResourceAction{
			Type:      resource.GetType(),
			Name:      resource.GetName(),
			Action:    ActionCreate,
			Message:   "would be created",
			Timestamp: now,
		})
	}

	// Add update actions
	for _, pair := range diff.ToUpdate {
		result.UpdatedResources = append(result.UpdatedResources, ResourceAction{
			Type:      pair.Desired.GetType(),
			Name:      pair.Desired.GetName(),
			Action:    ActionUpdate,
			Message:   "would be updated",
			Timestamp: now,
		})
	}

	// Add delete actions
	for _, resource := range diff.ToDelete {
		result.DeletedResources = append(result.DeletedResources, ResourceAction{
			Type:      resource.GetType(),
			Name:      resource.GetName(),
			Action:    ActionDelete,
			Message:   "would be deleted",
			Timestamp: now,
		})
	}
}

func (rc *DefaultReconciliationController) shouldCreate(resource Resource, toCreate []Resource) bool {
	for _, createResource := range toCreate {
		if createResource.GetName() == resource.GetName() && createResource.GetType() == resource.GetType() {
			return true
		}
	}
	return false
}

func (rc *DefaultReconciliationController) addError(result *ReconciliationResult, errorType ErrorType, resource ResourceReference, message string, cause error, recoverable bool) error {
	reconciliationError := NewReconciliationError(errorType, resource, message, cause, recoverable)
	result.Errors = append(result.Errors, reconciliationError)

	if !recoverable {
		return fmt.Errorf("%s", message)
	}

	return nil
}

// validateManifests performs comprehensive validation of input manifests
func (rc *DefaultReconciliationController) validateManifests(manifests []Resource) error {
	resourceNames := make(map[string]bool)

	for _, manifest := range manifests {
		// Check for duplicate names within the same type
		key := fmt.Sprintf("%s/%s", manifest.GetType(), manifest.GetName())
		if resourceNames[key] {
			return fmt.Errorf("duplicate resource found: %s", key)
		}
		resourceNames[key] = true

		// Validate resource name
		if manifest.GetName() == "" {
			return fmt.Errorf("resource name cannot be empty for type %s", manifest.GetType())
		}

		// Validate resource type
		if _, exists := rc.managers[manifest.GetType()]; !exists {
			return fmt.Errorf("unsupported resource type: %s", manifest.GetType())
		}
	}

	return nil
}

// buildDependencyGraphWithRetry builds dependency graph with retry logic
func (rc *DefaultReconciliationController) buildDependencyGraphWithRetry(ctx context.Context, manifests []Resource, result *ReconciliationResult) (*DependencyGraph, error) {
	const maxRetries = 3
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		dependencyGraph, err := rc.dependencyResolver.BuildDependencyGraph(manifests)
		if err == nil {
			return dependencyGraph, nil
		}

		lastErr = err
		if attempt < maxRetries {
			// Add a warning for retry attempts
			rc.addError(result, ErrorTypeDependency, ResourceReference{},
				fmt.Sprintf("dependency graph build attempt %d failed, retrying: %v", attempt, err), err, true)

			// Brief delay before retry
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(attempt) * 100 * time.Millisecond):
			}
		}
	}

	return nil, rc.addError(result, ErrorTypeDependency, ResourceReference{},
		fmt.Sprintf("failed to build dependency graph after %d attempts: %v", maxRetries, lastErr), lastErr, false)
}

// getCurrentStateWithRetry gets current state with retry and error recovery
func (rc *DefaultReconciliationController) getCurrentStateWithRetry(ctx context.Context, chartName string, result *ReconciliationResult) (map[ResourceType][]Resource, error) {
	actualStateByType := make(map[ResourceType][]Resource)
	const maxRetries = 3

	for resourceType, manager := range rc.managers {
		var lastErr error
		var actualResources []Resource

		for attempt := 1; attempt <= maxRetries; attempt++ {
			var err error
			actualResources, err = manager.GetActualState(ctx, chartName)
			if err == nil {
				break
			}

			lastErr = err
			if attempt < maxRetries {
				rc.addError(result, ErrorTypePodmanAPI, ResourceReference{Type: resourceType},
					fmt.Sprintf("failed to get actual state for %s (attempt %d), retrying: %v", resourceType, attempt, err), err, true)

				// Brief delay before retry
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
				}
			}
		}

		if lastErr != nil {
			rc.addError(result, ErrorTypePodmanAPI, ResourceReference{Type: resourceType},
				fmt.Sprintf("failed to get actual state for %s after %d attempts: %v", resourceType, maxRetries, lastErr), lastErr, true)
			// Continue with empty state for this resource type
			actualResources = make([]Resource, 0)
		}

		actualStateByType[resourceType] = actualResources
	}

	return actualStateByType, nil
}

// compareAllStatesWithValidation compares states with additional validation
func (rc *DefaultReconciliationController) compareAllStatesWithValidation(manifests []Resource, actualStateByType map[ResourceType][]Resource, result *ReconciliationResult) (*StateDiff, error) {
	// Group manifests by type
	manifestsByType := make(map[ResourceType][]Resource)
	for _, manifest := range manifests {
		resourceType := manifest.GetType()
		manifestsByType[resourceType] = append(manifestsByType[resourceType], manifest)
	}

	// Compare each resource type with error handling
	allDiff := &StateDiff{
		ToCreate:  make([]Resource, 0),
		ToUpdate:  make([]ResourcePair, 0),
		ToDelete:  make([]Resource, 0),
		Unchanged: make([]Resource, 0),
	}

	for resourceType := range rc.managers {
		desired := manifestsByType[resourceType]
		actual := actualStateByType[resourceType]

		diff, err := rc.stateComparator.CompareStates(desired, actual)
		if err != nil {
			rc.addError(result, ErrorTypeComparison, ResourceReference{Type: resourceType},
				fmt.Sprintf("failed to compare states for %s: %v", resourceType, err), err, true)
			continue
		}

		// Merge diffs
		allDiff.ToCreate = append(allDiff.ToCreate, diff.ToCreate...)
		allDiff.ToUpdate = append(allDiff.ToUpdate, diff.ToUpdate...)
		allDiff.ToDelete = append(allDiff.ToDelete, diff.ToDelete...)
		allDiff.Unchanged = append(allDiff.Unchanged, diff.Unchanged...)
	}

	return allDiff, nil
}

// executeReconciliationWithRecovery executes reconciliation with comprehensive error handling
func (rc *DefaultReconciliationController) executeReconciliationWithRecovery(ctx context.Context, result *ReconciliationResult, diff *StateDiff, creationOrder, deletionOrder [][]Resource) {
	// Execute creates in dependency order with error recovery
	for levelIndex, level := range creationOrder {
		rc.executeCreationLevel(ctx, result, level, diff.ToCreate, levelIndex)

		// Check if context was cancelled
		if ctx.Err() != nil {
			rc.addError(result, ErrorTypeConfiguration, ResourceReference{},
				"reconciliation cancelled by context", ctx.Err(), false)
			return
		}
	}

	// Execute updates with parallel processing where safe
	rc.executeUpdatesWithRecovery(ctx, result, diff.ToUpdate)

	// Execute deletes in reverse dependency order
	for levelIndex, level := range deletionOrder {
		rc.executeDeletionLevel(ctx, result, level, diff.ToDelete, levelIndex)

		// Check if context was cancelled
		if ctx.Err() != nil {
			rc.addError(result, ErrorTypeConfiguration, ResourceReference{},
				"reconciliation cancelled by context", ctx.Err(), false)
			return
		}
	}
}

// executeCreationLevel executes creation for a single dependency level
func (rc *DefaultReconciliationController) executeCreationLevel(ctx context.Context, result *ReconciliationResult, level []Resource, toCreate []Resource, levelIndex int) {
	for _, resource := range level {
		if rc.shouldCreate(resource, toCreate) {
			rc.executeCreateWithRetry(ctx, result, resource, levelIndex)
		}
	}
}

// executeDeletionLevel executes deletion for a single dependency level
func (rc *DefaultReconciliationController) executeDeletionLevel(ctx context.Context, result *ReconciliationResult, level []Resource, toDelete []Resource, levelIndex int) {
	for _, resource := range level {
		if rc.shouldDelete(resource, toDelete) {
			rc.executeDeleteWithRetry(ctx, result, resource, levelIndex)
		}
	}
}

// executeUpdatesWithRecovery executes updates with error recovery
func (rc *DefaultReconciliationController) executeUpdatesWithRecovery(ctx context.Context, result *ReconciliationResult, toUpdate []ResourcePair) {
	for _, pair := range toUpdate {
		rc.executeUpdateWithRetry(ctx, result, pair.Desired, pair.Actual)
	}
}

// executeCreateWithRetry creates a resource with retry logic
func (rc *DefaultReconciliationController) executeCreateWithRetry(ctx context.Context, result *ReconciliationResult, resource Resource, levelIndex int) {
	const maxRetries = 3
	startTime := time.Now()

	action := ResourceAction{
		Type:      resource.GetType(),
		Name:      resource.GetName(),
		Action:    ActionCreate,
		Timestamp: startTime,
	}

	manager, exists := rc.managers[resource.GetType()]
	if !exists {
		action.Error = fmt.Sprintf("no manager found for resource type %s", resource.GetType())
		action.Duration = time.Since(startTime)
		result.CreatedResources = append(result.CreatedResources, action)
		rc.addError(result, ErrorTypeConfiguration,
			ResourceReference{Type: resource.GetType(), Name: resource.GetName()},
			action.Error, nil, false)
		return
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := manager.CreateResource(ctx, resource)
		if err == nil {
			action.Duration = time.Since(startTime)
			action.Message = fmt.Sprintf("created successfully (level %d)", levelIndex)
			result.CreatedResources = append(result.CreatedResources, action)
			return
		}

		lastErr = err
		if attempt < maxRetries {
			// Brief delay before retry
			select {
			case <-ctx.Done():
				action.Error = "cancelled by context"
				action.Duration = time.Since(startTime)
				result.CreatedResources = append(result.CreatedResources, action)
				return
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}

	action.Error = fmt.Sprintf("failed after %d attempts: %v", maxRetries, lastErr)
	action.Duration = time.Since(startTime)
	result.CreatedResources = append(result.CreatedResources, action)
	rc.addError(result, ErrorTypePodmanAPI,
		ResourceReference{Type: resource.GetType(), Name: resource.GetName()},
		fmt.Sprintf("failed to create resource: %v", lastErr), lastErr, true)
}

// executeUpdateWithRetry updates a resource with retry logic
func (rc *DefaultReconciliationController) executeUpdateWithRetry(ctx context.Context, result *ReconciliationResult, desired, actual Resource) {
	const maxRetries = 3
	startTime := time.Now()

	action := ResourceAction{
		Type:      desired.GetType(),
		Name:      desired.GetName(),
		Action:    ActionUpdate,
		Timestamp: startTime,
	}

	manager, exists := rc.managers[desired.GetType()]
	if !exists {
		action.Error = fmt.Sprintf("no manager found for resource type %s", desired.GetType())
		action.Duration = time.Since(startTime)
		result.UpdatedResources = append(result.UpdatedResources, action)
		rc.addError(result, ErrorTypeConfiguration,
			ResourceReference{Type: desired.GetType(), Name: desired.GetName()},
			action.Error, nil, false)
		return
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := manager.UpdateResource(ctx, desired, actual)
		if err == nil {
			action.Duration = time.Since(startTime)
			action.Message = "updated successfully"
			result.UpdatedResources = append(result.UpdatedResources, action)
			return
		}

		lastErr = err
		if attempt < maxRetries {
			// Brief delay before retry
			select {
			case <-ctx.Done():
				action.Error = "cancelled by context"
				action.Duration = time.Since(startTime)
				result.UpdatedResources = append(result.UpdatedResources, action)
				return
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}

	action.Error = fmt.Sprintf("failed after %d attempts: %v", maxRetries, lastErr)
	action.Duration = time.Since(startTime)
	result.UpdatedResources = append(result.UpdatedResources, action)
	rc.addError(result, ErrorTypePodmanAPI,
		ResourceReference{Type: desired.GetType(), Name: desired.GetName()},
		fmt.Sprintf("failed to update resource: %v", lastErr), lastErr, true)
}

// executeDeleteWithRetry deletes a resource with retry logic
func (rc *DefaultReconciliationController) executeDeleteWithRetry(ctx context.Context, result *ReconciliationResult, resource Resource, levelIndex int) {
	const maxRetries = 3
	startTime := time.Now()

	action := ResourceAction{
		Type:      resource.GetType(),
		Name:      resource.GetName(),
		Action:    ActionDelete,
		Timestamp: startTime,
	}

	manager, exists := rc.managers[resource.GetType()]
	if !exists {
		action.Error = fmt.Sprintf("no manager found for resource type %s", resource.GetType())
		action.Duration = time.Since(startTime)
		result.DeletedResources = append(result.DeletedResources, action)
		rc.addError(result, ErrorTypeConfiguration,
			ResourceReference{Type: resource.GetType(), Name: resource.GetName()},
			action.Error, nil, false)
		return
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := manager.DeleteResource(ctx, resource)
		if err == nil {
			action.Duration = time.Since(startTime)
			action.Message = fmt.Sprintf("deleted successfully (level %d)", levelIndex)
			result.DeletedResources = append(result.DeletedResources, action)
			return
		}

		lastErr = err
		if attempt < maxRetries {
			// Brief delay before retry
			select {
			case <-ctx.Done():
				action.Error = "cancelled by context"
				action.Duration = time.Since(startTime)
				result.DeletedResources = append(result.DeletedResources, action)
				return
			case <-time.After(time.Duration(attempt) * 500 * time.Millisecond):
			}
		}
	}

	action.Error = fmt.Sprintf("failed after %d attempts: %v", maxRetries, lastErr)
	action.Duration = time.Since(startTime)
	result.DeletedResources = append(result.DeletedResources, action)
	rc.addError(result, ErrorTypePodmanAPI,
		ResourceReference{Type: resource.GetType(), Name: resource.GetName()},
		fmt.Sprintf("failed to delete resource: %v", lastErr), lastErr, true)
}

// cleanupOrphanedResourcesWithRecovery removes orphaned resources with error handling
func (rc *DefaultReconciliationController) cleanupOrphanedResourcesWithRecovery(ctx context.Context, result *ReconciliationResult, manifests []Resource, actualStateByType map[ResourceType][]Resource, deletionOrder [][]Resource) {
	// Create a set of desired resource names by type
	desiredByType := make(map[ResourceType]map[string]bool)
	for _, manifest := range manifests {
		resourceType := manifest.GetType()
		if desiredByType[resourceType] == nil {
			desiredByType[resourceType] = make(map[string]bool)
		}
		desiredByType[resourceType][manifest.GetName()] = true
	}

	// Find orphaned resources and organize by dependency levels
	orphanedByLevel := make([][]Resource, len(deletionOrder))

	for levelIndex, level := range deletionOrder {
		for _, levelResource := range level {
			resourceType := levelResource.GetType()
			actualResources := actualStateByType[resourceType]
			desired := desiredByType[resourceType]
			if desired == nil {
				desired = make(map[string]bool)
			}

			for _, actualResource := range actualResources {
				if !desired[actualResource.GetName()] {
					// Check if this resource matches the current level
					if rc.resourceMatchesLevel(actualResource, level) {
						orphanedByLevel[levelIndex] = append(orphanedByLevel[levelIndex], actualResource)
					}
				}
			}
		}
	}

	// Delete orphaned resources in proper dependency order
	for levelIndex, orphanedResources := range orphanedByLevel {
		for _, orphanedResource := range orphanedResources {
			rc.executeDeleteWithRetry(ctx, result, orphanedResource, levelIndex)

			// Check if context was cancelled
			if ctx.Err() != nil {
				return
			}
		}
	}
}

// updateReconciliationStatus updates the internal status tracking
func (rc *DefaultReconciliationController) updateReconciliationStatus(chartName string, result *ReconciliationResult, startTime time.Time) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	status := &ReconciliationStatus{
		ChartName:      chartName,
		LastReconciled: startTime,
		ResourceCounts: make(map[string]int),
		Errors:         result.Errors,
	}

	// Count successful operations
	successfulCreates := 0
	successfulUpdates := 0
	successfulDeletes := 0

	for _, action := range result.CreatedResources {
		if action.Error == "" {
			successfulCreates++
		}
	}

	for _, action := range result.UpdatedResources {
		if action.Error == "" {
			successfulUpdates++
		}
	}

	for _, action := range result.DeletedResources {
		if action.Error == "" {
			successfulDeletes++
		}
	}

	status.ResourceCounts["created"] = successfulCreates
	status.ResourceCounts["updated"] = successfulUpdates
	status.ResourceCounts["deleted"] = successfulDeletes

	// Determine overall status
	if len(result.Errors) == 0 {
		status.Status = "healthy"
	} else {
		nonRecoverableErrors := 0
		for _, err := range result.Errors {
			if !err.Recoverable {
				nonRecoverableErrors++
			}
		}
		if nonRecoverableErrors > 0 {
			status.Status = "failed"
		} else {
			status.Status = "degraded"
		}
	}

	rc.lastStatus[chartName] = status
}

// Helper methods

func (rc *DefaultReconciliationController) shouldDelete(resource Resource, toDelete []Resource) bool {
	for _, deleteResource := range toDelete {
		if deleteResource.GetName() == resource.GetName() && deleteResource.GetType() == resource.GetType() {
			return true
		}
	}
	return false
}

func (rc *DefaultReconciliationController) resourceMatchesLevel(resource Resource, level []Resource) bool {
	for _, levelResource := range level {
		if resource.GetName() == levelResource.GetName() && resource.GetType() == levelResource.GetType() {
			return true
		}
	}
	return false
}

func (rc *DefaultReconciliationController) generateSummary(result *ReconciliationResult) string {
	created := len(result.CreatedResources)
	updated := len(result.UpdatedResources)
	deleted := len(result.DeletedResources)
	errors := len(result.Errors)

	// Count successful operations
	successfulCreates := 0
	successfulUpdates := 0
	successfulDeletes := 0

	for _, action := range result.CreatedResources {
		if action.Error == "" {
			successfulCreates++
		}
	}

	for _, action := range result.UpdatedResources {
		if action.Error == "" {
			successfulUpdates++
		}
	}

	for _, action := range result.DeletedResources {
		if action.Error == "" {
			successfulDeletes++
		}
	}

	if errors > 0 {
		return fmt.Sprintf("Reconciliation completed with errors: %d/%d created, %d/%d updated, %d/%d deleted, %d errors",
			successfulCreates, created, successfulUpdates, updated, successfulDeletes, deleted, errors)
	}

	return fmt.Sprintf("Reconciliation completed successfully: %d created, %d updated, %d deleted",
		created, updated, deleted)
}
