package resource

import (
	"context"
	"cutepod/internal/podman"
	"encoding/base64"
	"testing"
)

func TestSecretManager_GetResourceType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	if manager.GetResourceType() != ResourceTypeSecret {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeSecret, manager.GetResourceType())
	}
}

func TestSecretManager_GetDesiredState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// Create test manifests
	secret1 := NewSecretResource()
	secret1.ObjectMeta.Name = "secret1"
	secret1.Spec.Type = SecretTypeOpaque
	secret1.Spec.Data = map[string]string{
		"key1": base64.StdEncoding.EncodeToString([]byte("value1")),
	}

	secret2 := NewSecretResource()
	secret2.ObjectMeta.Name = "secret2"
	secret2.Spec.Type = SecretTypeOpaque
	secret2.Spec.Data = map[string]string{
		"key2": base64.StdEncoding.EncodeToString([]byte("value2")),
	}

	container := NewContainerResource()
	container.ObjectMeta.Name = "test-container"

	manifests := []Resource{secret1, container, secret2}

	// Test GetDesiredState
	secrets, err := manager.GetDesiredState(manifests)
	if err != nil {
		t.Fatalf("GetDesiredState failed: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(secrets))
	}
	if secrets[0].GetName() != "secret1" {
		t.Errorf("Expected first secret name 'secret1', got '%s'", secrets[0].GetName())
	}
	if secrets[1].GetName() != "secret2" {
		t.Errorf("Expected second secret name 'secret2', got '%s'", secrets[1].GetName())
	}
}

func TestSecretManager_GetActualState(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// Add mock secrets to the client
	secret1 := &podman.SecretInfo{
		ID:   "secret1-id",
		Name: "secret1",
		Labels: map[string]string{
			"cutepod.Namespace": "test-namespace",
			"cutepod.Chart":     "test-chart",
		},
	}
	secret2 := &podman.SecretInfo{
		ID:   "secret2-id",
		Name: "secret2",
		Labels: map[string]string{
			"cutepod.Namespace": "test-namespace",
			"cutepod.Chart":     "test-chart",
		},
	}

	// Manually add secrets to mock client's internal storage
	ctx := context.Background()
	_, _ = mockClient.CreateSecret(ctx, podman.SecretSpec{
		Name:   secret1.Name,
		Data:   []byte("mock-data"),
		Labels: secret1.Labels,
	})
	_, _ = mockClient.CreateSecret(ctx, podman.SecretSpec{
		Name:   secret2.Name,
		Data:   []byte("mock-data"),
		Labels: secret2.Labels,
	})

	// Test GetActualState
	secrets, err := manager.GetActualState(context.Background(), "test-namespace")
	if err != nil {
		t.Fatalf("GetActualState failed: %v", err)
	}
	if len(secrets) != 2 {
		t.Errorf("Expected 2 secrets, got %d", len(secrets))
	}

	// Verify first secret
	secret1Resource := secrets[0].(*SecretResource)
	if secret1Resource.GetName() != "secret1" {
		t.Errorf("Expected first secret name 'secret1', got '%s'", secret1Resource.GetName())
	}
	if secret1Resource.GetNamespace() != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", secret1Resource.GetNamespace())
	}
	if secret1Resource.Spec.Type != SecretTypeOpaque {
		t.Errorf("Expected secret type %s, got %s", SecretTypeOpaque, secret1Resource.Spec.Type)
	}
	if secret1Resource.GetLabels()["cutepod.Chart"] != "test-chart" {
		t.Errorf("Expected chart label 'test-chart', got '%s'", secret1Resource.GetLabels()["cutepod.Chart"])
	}

	// Verify second secret
	secret2Resource := secrets[1].(*SecretResource)
	if secret2Resource.GetName() != "secret2" {
		t.Errorf("Expected second secret name 'secret2', got '%s'", secret2Resource.GetName())
	}
	if secret2Resource.GetNamespace() != "test-namespace" {
		t.Errorf("Expected namespace 'test-namespace', got '%s'", secret2Resource.GetNamespace())
	}
}

func TestSecretManager_CreateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// Create test secret
	secret := NewSecretResource()
	secret.ObjectMeta.Name = "test-secret"
	secret.ObjectMeta.Namespace = "test-namespace"
	secret.Spec.Type = SecretTypeOpaque
	secret.Spec.Data = map[string]string{
		"username": base64.StdEncoding.EncodeToString([]byte("admin")),
		"password": base64.StdEncoding.EncodeToString([]byte("secret123")),
	}
	secret.SetLabels(map[string]string{
		"cutepod.Namespace": "test-namespace",
		"cutepod.Chart":     "test-chart",
	})

	// Test CreateResource
	err := manager.CreateResource(context.Background(), secret)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the secret was created in the mock client
	if mockClient.GetCallCount("CreateSecret") != 1 {
		t.Errorf("Expected CreateSecret to be called once, got %d calls", mockClient.GetCallCount("CreateSecret"))
	}

	// Verify the secret exists in the mock client
	secrets, err := mockClient.ListSecrets(context.Background(), map[string][]string{
		"label": {"cutepod.Namespace=test-namespace"},
	})
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}
	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].Name != "test-secret" {
		t.Errorf("Expected secret name 'test-secret', got '%s'", secrets[0].Name)
	}
}

func TestSecretManager_CreateResource_SingleKey(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// Create test secret with single key
	secret := NewSecretResource()
	secret.ObjectMeta.Name = "single-key-secret"
	secret.Spec.Data = map[string]string{
		"token": base64.StdEncoding.EncodeToString([]byte("abc123")),
	}

	// Test CreateResource
	err := manager.CreateResource(context.Background(), secret)
	if err != nil {
		t.Fatalf("CreateResource failed: %v", err)
	}

	// Verify the secret was created
	if mockClient.GetCallCount("CreateSecret") != 1 {
		t.Errorf("Expected CreateSecret to be called once, got %d calls", mockClient.GetCallCount("CreateSecret"))
	}

	// Verify the secret exists
	secrets, err := mockClient.ListSecrets(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}
	if len(secrets) != 1 {
		t.Errorf("Expected 1 secret, got %d", len(secrets))
	}
	if secrets[0].Name != "single-key-secret" {
		t.Errorf("Expected secret name 'single-key-secret', got '%s'", secrets[0].Name)
	}
}

func TestSecretManager_UpdateResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// First create the existing secret
	ctx := context.Background()
	_, _ = mockClient.CreateSecret(ctx, podman.SecretSpec{
		Name: "test-secret",
		Data: []byte("old-value"),
	})

	// Create desired secret
	desired := NewSecretResource()
	desired.ObjectMeta.Name = "test-secret"
	desired.Spec.Data = map[string]string{
		"key": base64.StdEncoding.EncodeToString([]byte("new-value")),
	}

	// Create actual secret
	actual := NewSecretResource()
	actual.ObjectMeta.Name = "test-secret"
	actual.Spec.Data = map[string]string{
		"key": base64.StdEncoding.EncodeToString([]byte("old-value")),
	}

	// Test UpdateResource
	err := manager.UpdateResource(context.Background(), desired, actual)
	if err != nil {
		t.Fatalf("UpdateResource failed: %v", err)
	}

	// Verify UpdateSecret was called
	if mockClient.GetCallCount("UpdateSecret") != 1 {
		t.Errorf("Expected UpdateSecret to be called once, got %d calls", mockClient.GetCallCount("UpdateSecret"))
	}
}

func TestSecretManager_DeleteResource(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	// First create the secret
	ctx := context.Background()
	_, _ = mockClient.CreateSecret(ctx, podman.SecretSpec{
		Name: "test-secret",
		Data: []byte("test-data"),
	})

	// Create test secret resource
	secret := NewSecretResource()
	secret.ObjectMeta.Name = "test-secret"

	// Test DeleteResource
	err := manager.DeleteResource(context.Background(), secret)
	if err != nil {
		t.Fatalf("DeleteResource failed: %v", err)
	}

	// Verify RemoveSecret was called
	if mockClient.GetCallCount("RemoveSecret") != 1 {
		t.Errorf("Expected RemoveSecret to be called once, got %d calls", mockClient.GetCallCount("RemoveSecret"))
	}

	// Verify the secret was removed
	secrets, err := mockClient.ListSecrets(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}
	if len(secrets) != 0 {
		t.Errorf("Expected 0 secrets after deletion, got %d", len(secrets))
	}
}

func TestSecretManager_CompareResources(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	tests := []struct {
		name     string
		desired  *SecretResource
		actual   *SecretResource
		expected bool
	}{
		{
			name: "identical secrets",
			desired: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			actual: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			expected: true,
		},
		{
			name: "different data",
			desired: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
					},
				},
			},
			actual: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "different-value",
					},
				},
			},
			expected: false,
		},
		{
			name: "different number of keys",
			desired: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			actual: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
					},
				},
			},
			expected: false,
		},
		{
			name: "missing key",
			desired: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
			actual: &SecretResource{
				Spec: CuteSecretSpec{
					Type: SecretTypeOpaque,
					Data: map[string]string{
						"key1": "value1",
						"key3": "value3",
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, err := manager.CompareResources(tt.desired, tt.actual)
			if err != nil {
				t.Errorf("CompareResources failed: %v", err)
			}
			if match != tt.expected {
				t.Errorf("Expected match=%v, got match=%v", tt.expected, match)
			}
		})
	}
}

func TestSecretManager_CompareResources_InvalidTypes(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	secret := NewSecretResource()
	container := NewContainerResource()

	// Test with invalid desired type
	_, err := manager.CompareResources(container, secret)
	if err == nil {
		t.Error("Expected error for invalid desired type")
	}
	if err != nil && !contains(err.Error(), "expected SecretResource for desired") {
		t.Errorf("Expected error message to contain 'expected SecretResource for desired', got: %v", err)
	}

	// Test with invalid actual type
	_, err = manager.CompareResources(secret, container)
	if err == nil {
		t.Error("Expected error for invalid actual type")
	}
	if err != nil && !contains(err.Error(), "expected SecretResource for actual") {
		t.Errorf("Expected error message to contain 'expected SecretResource for actual', got: %v", err)
	}
}

func TestSecretManager_CreateResource_InvalidType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	container := NewContainerResource()

	err := manager.CreateResource(context.Background(), container)
	if err == nil {
		t.Error("Expected error for invalid resource type")
	}
	if err != nil && !contains(err.Error(), "expected SecretResource") {
		t.Errorf("Expected error message to contain 'expected SecretResource', got: %v", err)
	}
}

func TestSecretManager_UpdateResource_InvalidType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	secret := NewSecretResource()
	container := NewContainerResource()

	err := manager.UpdateResource(context.Background(), container, secret)
	if err == nil {
		t.Error("Expected error for invalid desired resource type")
	}
	if err != nil && !contains(err.Error(), "expected SecretResource for desired") {
		t.Errorf("Expected error message to contain 'expected SecretResource for desired', got: %v", err)
	}
}

func TestSecretManager_DeleteResource_InvalidType(t *testing.T) {
	mockClient := podman.NewMockPodmanClient()
	manager := NewSecretManager(mockClient)

	container := NewContainerResource()

	err := manager.DeleteResource(context.Background(), container)
	if err == nil {
		t.Error("Expected error for invalid resource type")
	}
	if err != nil && !contains(err.Error(), "expected SecretResource") {
		t.Errorf("Expected error message to contain 'expected SecretResource', got: %v", err)
	}
}

func TestSecretResource_GetDecodedData(t *testing.T) {
	secret := NewSecretResource()
	secret.Spec.Data = map[string]string{
		"username": base64.StdEncoding.EncodeToString([]byte("admin")),
		"password": base64.StdEncoding.EncodeToString([]byte("secret123")),
	}

	decoded, err := secret.GetDecodedData()
	if err != nil {
		t.Fatalf("GetDecodedData failed: %v", err)
	}
	if string(decoded["username"]) != "admin" {
		t.Errorf("Expected username 'admin', got '%s'", string(decoded["username"]))
	}
	if string(decoded["password"]) != "secret123" {
		t.Errorf("Expected password 'secret123', got '%s'", string(decoded["password"]))
	}
}

func TestSecretResource_GetDecodedData_InvalidBase64(t *testing.T) {
	secret := NewSecretResource()
	secret.Spec.Data = map[string]string{
		"invalid": "not-base64!@#",
	}

	_, err := secret.GetDecodedData()
	if err == nil {
		t.Error("Expected error for invalid base64 data")
	}
	if err != nil && !contains(err.Error(), "failed to decode base64 data") {
		t.Errorf("Expected error message to contain 'failed to decode base64 data', got: %v", err)
	}
}

func TestSecretResource_SetData(t *testing.T) {
	secret := NewSecretResource()

	data := map[string][]byte{
		"username": []byte("admin"),
		"password": []byte("secret123"),
	}

	secret.SetData(data)

	expectedUsername := base64.StdEncoding.EncodeToString([]byte("admin"))
	expectedPassword := base64.StdEncoding.EncodeToString([]byte("secret123"))

	if secret.Spec.Data["username"] != expectedUsername {
		t.Errorf("Expected username '%s', got '%s'", expectedUsername, secret.Spec.Data["username"])
	}
	if secret.Spec.Data["password"] != expectedPassword {
		t.Errorf("Expected password '%s', got '%s'", expectedPassword, secret.Spec.Data["password"])
	}
}
