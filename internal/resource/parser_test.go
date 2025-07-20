package resource

import (
	"testing"
)

func TestManifestParser_ParseNetwork(t *testing.T) {
	parser := NewManifestParser()

	// Test valid CuteNetwork manifest
	networkYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: test-network
  labels:
    app: test-app
spec:
  driver: bridge
  subnet: "172.20.0.0/16"
  gateway: "172.20.0.1"
  options:
    mtu: "1500"
    com.docker.network.bridge.enable_icc: "true"
`

	err := parser.ParseManifest([]byte(networkYAML))
	if err != nil {
		t.Fatalf("Failed to parse network manifest: %v", err)
	}

	registry := parser.GetRegistry()
	resources := registry.GetResourcesByType(ResourceTypeNetwork)

	if len(resources) != 1 {
		t.Fatalf("Expected 1 network resource, got %d", len(resources))
	}

	network := resources[0].(*NetworkResource)
	if network.GetName() != "test-network" {
		t.Errorf("Expected network name 'test-network', got '%s'", network.GetName())
	}

	if network.Spec.Driver != "bridge" {
		t.Errorf("Expected driver 'bridge', got '%s'", network.Spec.Driver)
	}

	if network.Spec.Subnet != "172.20.0.0/16" {
		t.Errorf("Expected subnet '172.20.0.0/16', got '%s'", network.Spec.Subnet)
	}

	if network.Spec.Gateway != "172.20.0.1" {
		t.Errorf("Expected gateway '172.20.0.1', got '%s'", network.Spec.Gateway)
	}

	if network.Spec.Options["mtu"] != "1500" {
		t.Errorf("Expected mtu option '1500', got '%s'", network.Spec.Options["mtu"])
	}

	labels := network.GetLabels()
	if labels["app"] != "test-app" {
		t.Errorf("Expected app label 'test-app', got '%s'", labels["app"])
	}
}

func TestManifestParser_ParseNetwork_DefaultDriver(t *testing.T) {
	parser := NewManifestParser()

	// Test CuteNetwork manifest without driver (should default to bridge)
	networkYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: default-network
spec:
  subnet: "172.21.0.0/16"
`

	err := parser.ParseManifest([]byte(networkYAML))
	if err != nil {
		t.Fatalf("Failed to parse network manifest: %v", err)
	}

	registry := parser.GetRegistry()
	resources := registry.GetResourcesByType(ResourceTypeNetwork)

	if len(resources) != 1 {
		t.Fatalf("Expected 1 network resource, got %d", len(resources))
	}

	network := resources[0].(*NetworkResource)
	if network.Spec.Driver != "bridge" {
		t.Errorf("Expected default driver 'bridge', got '%s'", network.Spec.Driver)
	}
}

func TestManifestParser_ParseNetwork_MinimalSpec(t *testing.T) {
	parser := NewManifestParser()

	// Test minimal CuteNetwork manifest
	networkYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: minimal-network
spec: {}
`

	err := parser.ParseManifest([]byte(networkYAML))
	if err != nil {
		t.Fatalf("Failed to parse minimal network manifest: %v", err)
	}

	registry := parser.GetRegistry()
	resources := registry.GetResourcesByType(ResourceTypeNetwork)

	if len(resources) != 1 {
		t.Fatalf("Expected 1 network resource, got %d", len(resources))
	}

	network := resources[0].(*NetworkResource)
	if network.GetName() != "minimal-network" {
		t.Errorf("Expected network name 'minimal-network', got '%s'", network.GetName())
	}

	if network.Spec.Driver != "bridge" {
		t.Errorf("Expected default driver 'bridge', got '%s'", network.Spec.Driver)
	}
}

func TestManifestParser_ParseNetwork_ValidationErrors(t *testing.T) {
	parser := NewManifestParser()

	// Test network without name
	networkYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata: {}
spec:
  driver: bridge
`

	err := parser.ParseManifest([]byte(networkYAML))
	if err == nil {
		t.Error("Expected error for network without name")
	}
}

func TestManifestParser_ParseNetwork_InvalidYAML(t *testing.T) {
	parser := NewManifestParser()

	// Test invalid YAML
	networkYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: test-network
spec:
  driver: bridge
  options:
    - invalid: yaml: structure
`

	err := parser.ParseManifest([]byte(networkYAML))
	if err == nil {
		t.Error("Expected error for invalid YAML")
	}
}

func TestManifestParser_ParseMultipleNetworks(t *testing.T) {
	parser := NewManifestParser()

	// Test multiple networks in one manifest
	networksYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: network-1
spec:
  driver: bridge
  subnet: "172.20.0.0/16"
---
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: network-2
spec:
  driver: macvlan
  subnet: "172.21.0.0/16"
`

	err := parser.ParseManifest([]byte(networksYAML))
	if err != nil {
		t.Fatalf("Failed to parse multiple network manifests: %v", err)
	}

	registry := parser.GetRegistry()
	resources := registry.GetResourcesByType(ResourceTypeNetwork)

	if len(resources) != 2 {
		t.Fatalf("Expected 2 network resources, got %d", len(resources))
	}

	// Check both networks were parsed correctly
	networkNames := make(map[string]*NetworkResource)
	for _, res := range resources {
		network := res.(*NetworkResource)
		networkNames[network.GetName()] = network
	}

	if network1, exists := networkNames["network-1"]; exists {
		if network1.Spec.Driver != "bridge" {
			t.Errorf("Expected network-1 driver 'bridge', got '%s'", network1.Spec.Driver)
		}
		if network1.Spec.Subnet != "172.20.0.0/16" {
			t.Errorf("Expected network-1 subnet '172.20.0.0/16', got '%s'", network1.Spec.Subnet)
		}
	} else {
		t.Error("network-1 not found")
	}

	if network2, exists := networkNames["network-2"]; exists {
		if network2.Spec.Driver != "macvlan" {
			t.Errorf("Expected network-2 driver 'macvlan', got '%s'", network2.Spec.Driver)
		}
		if network2.Spec.Subnet != "172.21.0.0/16" {
			t.Errorf("Expected network-2 subnet '172.21.0.0/16', got '%s'", network2.Spec.Subnet)
		}
	} else {
		t.Error("network-2 not found")
	}
}

func TestManifestParser_ParseMixedResources(t *testing.T) {
	parser := NewManifestParser()

	// Test mixed resources including networks
	mixedYAML := `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: web-network
spec:
  driver: bridge
  subnet: "172.20.0.0/16"
---
apiVersion: cutepod/v1alpha0
kind: CuteContainer
metadata:
  name: web-server
spec:
  image: nginx:latest
  networks:
    - web-network
`

	err := parser.ParseManifest([]byte(mixedYAML))
	if err != nil {
		t.Fatalf("Failed to parse mixed manifests: %v", err)
	}

	registry := parser.GetRegistry()

	networks := registry.GetResourcesByType(ResourceTypeNetwork)
	if len(networks) != 1 {
		t.Errorf("Expected 1 network resource, got %d", len(networks))
	}

	containers := registry.GetResourcesByType(ResourceTypeContainer)
	if len(containers) != 1 {
		t.Errorf("Expected 1 container resource, got %d", len(containers))
	}

	// Verify the container references the network
	container := containers[0].(*ContainerResource)
	if len(container.Spec.Networks) != 1 || container.Spec.Networks[0] != "web-network" {
		t.Error("Container should reference web-network")
	}
}

func TestManifestParser_NetworkValidation(t *testing.T) {
	tests := []struct {
		name        string
		yaml        string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid network",
			yaml: `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata:
  name: valid-network
spec:
  driver: bridge
`,
			expectError: false,
		},
		{
			name: "network without name",
			yaml: `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
metadata: {}
spec:
  driver: bridge
`,
			expectError: true,
			errorMsg:    "network name cannot be empty",
		},
		{
			name: "network with empty metadata",
			yaml: `
apiVersion: cutepod/v1alpha0
kind: CuteNetwork
spec:
  driver: bridge
`,
			expectError: true,
			errorMsg:    "network name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewManifestParser()
			err := parser.ParseManifest([]byte(tt.yaml))

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 0; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
