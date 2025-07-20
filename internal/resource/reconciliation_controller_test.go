package resource

import (
	"testing"
	"time"
)

func TestReconciliationResult_Summary(t *testing.T) {
	controller := &DefaultReconciliationController{}

	result := &ReconciliationResult{
		CreatedResources: []ResourceAction{
			{Type: ResourceTypeContainer, Name: "test1", Action: ActionCreate},
			{Type: ResourceTypeNetwork, Name: "test2", Action: ActionCreate},
		},
		UpdatedResources: []ResourceAction{
			{Type: ResourceTypeContainer, Name: "test3", Action: ActionUpdate},
		},
		DeletedResources: []ResourceAction{},
		Errors:           []*ReconciliationError{},
		Duration:         100 * time.Millisecond,
	}

	summary := controller.generateSummary(result)

	expected := "Reconciliation completed successfully: 2 created, 1 updated, 0 deleted"
	if summary != expected {
		t.Errorf("Expected summary '%s', got '%s'", expected, summary)
	}
}

func TestReconciliationResult_SummaryWithErrors(t *testing.T) {
	controller := &DefaultReconciliationController{}

	result := &ReconciliationResult{
		CreatedResources: []ResourceAction{
			{Type: ResourceTypeContainer, Name: "test1", Action: ActionCreate},
		},
		UpdatedResources: []ResourceAction{},
		DeletedResources: []ResourceAction{},
		Errors: []*ReconciliationError{
			NewPodmanAPIError(ResourceReference{Type: ResourceTypeContainer, Name: "test"}, "Test error", nil, true),
		},
		Duration: 100 * time.Millisecond,
	}

	summary := controller.generateSummary(result)

	expected := "Reconciliation completed with errors: 1/1 created, 0/0 updated, 0/0 deleted, 1 errors"
	if summary != expected {
		t.Errorf("Expected summary '%s', got '%s'", expected, summary)
	}
}
