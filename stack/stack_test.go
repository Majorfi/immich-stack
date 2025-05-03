package stack

import (
	"testing"
)

func TestModifyStack(t *testing.T) {
	// Create a test stack
	stack := &Stack{
		Services: map[string]Service{
			"test-service": {
				Name:  "test-service",
				Image: "test-image:latest",
				Ports: []Port{
					{
						Target:    8080,
						Published: "8080",
						Protocol:  "tcp",
						Mode:      "host",
					},
				},
				Environment: []string{
					"TEST_ENV=value",
				},
				Volumes: []Volume{
					{
						Type:   "bind",
						Source: "/host/path",
						Target: "/container/path",
					},
				},
			},
		},
	}

	// Test modifying the stack
	modifications := map[string]interface{}{
		"services.test-service.image": "new-image:latest",
		"services.test-service.ports": []Port{
			{
				Target:    9090,
				Published: "9090",
				Protocol:  "tcp",
				Mode:      "host",
			},
		},
		"services.test-service.environment": []string{
			"NEW_ENV=new-value",
		},
		"services.test-service.volumes": []Volume{
			{
				Type:   "bind",
				Source: "/new/host/path",
				Target: "/new/container/path",
			},
		},
	}

	modifiedStack, err := ModifyStack(stack, modifications)
	if err != nil {
		t.Fatalf("Failed to modify stack: %v", err)
	}

	// Verify modifications
	service := modifiedStack.Services["test-service"]
	if service.Image != "new-image:latest" {
		t.Errorf("Expected image to be 'new-image:latest', got '%s'", service.Image)
	}

	if len(service.Ports) != 1 || service.Ports[0].Target != 9090 {
		t.Errorf("Expected port target to be 9090, got %d", service.Ports[0].Target)
	}

	if len(service.Environment) != 1 || service.Environment[0] != "NEW_ENV=new-value" {
		t.Errorf("Expected environment to be ['NEW_ENV=new-value'], got %v", service.Environment)
	}

	if len(service.Volumes) != 1 || service.Volumes[0].Source != "/new/host/path" {
		t.Errorf("Expected volume source to be '/new/host/path', got '%s'", service.Volumes[0].Source)
	}
}

func TestApplyCriteria(t *testing.T) {
	// Create a test stack
	stack := &Stack{
		Services: map[string]Service{
			"test-service": {
				Name:  "test-service",
				Image: "test-image:latest",
				Ports: []Port{
					{
						Target:    8080,
						Published: "8080",
						Protocol:  "tcp",
						Mode:      "host",
					},
				},
				Environment: []string{
					"TEST_ENV=value",
				},
				Volumes: []Volume{
					{
						Type:   "bind",
						Source: "/host/path",
						Target: "/container/path",
					},
				},
			},
		},
	}

	// Test applying criteria
	criteria := map[string]interface{}{
		"services.test-service.image": "new-image:latest",
		"services.test-service.ports": []Port{
			{
				Target:    9090,
				Published: "9090",
				Protocol:  "tcp",
				Mode:      "host",
			},
		},
		"services.test-service.environment": []string{
			"NEW_ENV=new-value",
		},
		"services.test-service.volumes": []Volume{
			{
				Type:   "bind",
				Source: "/new/host/path",
				Target: "/new/container/path",
			},
		},
	}

	modifiedStack, err := ApplyCriteria(stack, criteria)
	if err != nil {
		t.Fatalf("Failed to apply criteria: %v", err)
	}

	// Verify modifications
	service := modifiedStack.Services["test-service"]
	if service.Image != "new-image:latest" {
		t.Errorf("Expected image to be 'new-image:latest', got '%s'", service.Image)
	}

	if len(service.Ports) != 1 || service.Ports[0].Target != 9090 {
		t.Errorf("Expected port target to be 9090, got %d", service.Ports[0].Target)
	}

	if len(service.Environment) != 1 || service.Environment[0] != "NEW_ENV=new-value" {
		t.Errorf("Expected environment to be ['NEW_ENV=new-value'], got %v", service.Environment)
	}

	if len(service.Volumes) != 1 || service.Volumes[0].Source != "/new/host/path" {
		t.Errorf("Expected volume source to be '/new/host/path', got '%s'", service.Volumes[0].Source)
	}
}
