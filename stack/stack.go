package stack

import (
	"fmt"
	"strings"
)

// Stack represents a Docker Compose stack configuration
type Stack struct {
	Services map[string]Service `yaml:"services"`
}

// Service represents a Docker service configuration
type Service struct {
	Name            string            `yaml:"name"`
	Image           string            `yaml:"image"`
	Ports           []Port            `yaml:"ports,omitempty"`
	Environment     []string          `yaml:"environment,omitempty"`
	Volumes         []Volume          `yaml:"volumes,omitempty"`
	Labels          map[string]string `yaml:"labels,omitempty"`
	Networks        []string          `yaml:"networks,omitempty"`
	DependsOn       []string          `yaml:"depends_on,omitempty"`
	Restart         string            `yaml:"restart,omitempty"`
	HealthCheck     *HealthCheck      `yaml:"healthcheck,omitempty"`
	Deploy          *Deploy           `yaml:"deploy,omitempty"`
	Command         []string          `yaml:"command,omitempty"`
	Entrypoint      []string          `yaml:"entrypoint,omitempty"`
	User            string            `yaml:"user,omitempty"`
	WorkingDir      string            `yaml:"working_dir,omitempty"`
	DomainName      string            `yaml:"domainname,omitempty"`
	Hostname        string            `yaml:"hostname,omitempty"`
	MacAddress      string            `yaml:"mac_address,omitempty"`
	IPc             string            `yaml:"ipc,omitempty"`
	Privileged      bool              `yaml:"privileged,omitempty"`
	ReadOnly        bool              `yaml:"read_only,omitempty"`
	ShmSize         string            `yaml:"shm_size,omitempty"`
	StdinOpen       bool              `yaml:"stdin_open,omitempty"`
	Tty             bool              `yaml:"tty,omitempty"`
	SecurityOpt     []string          `yaml:"security_opt,omitempty"`
	StopSignal      string            `yaml:"stop_signal,omitempty"`
	StopGracePeriod string            `yaml:"stop_grace_period,omitempty"`
	Sysctls         map[string]string `yaml:"sysctls,omitempty"`
	Ulimits         map[string]Ulimit `yaml:"ulimits,omitempty"`
	Isolation       string            `yaml:"isolation,omitempty"`
}

// Port represents a port mapping configuration
type Port struct {
	Target    int    `yaml:"target"`
	Published string `yaml:"published"`
	Protocol  string `yaml:"protocol"`
	Mode      string `yaml:"mode"`
}

// Volume represents a volume mapping configuration
type Volume struct {
	Type   string `yaml:"type"`
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// HealthCheck represents a health check configuration
type HealthCheck struct {
	Test        []string `yaml:"test"`
	Interval    string   `yaml:"interval,omitempty"`
	Timeout     string   `yaml:"timeout,omitempty"`
	Retries     int      `yaml:"retries,omitempty"`
	StartPeriod string   `yaml:"start_period,omitempty"`
}

// Deploy represents deployment configuration
type Deploy struct {
	Mode           string          `yaml:"mode,omitempty"`
	Replicas       int             `yaml:"replicas,omitempty"`
	Placement      *Placement      `yaml:"placement,omitempty"`
	UpdateConfig   *UpdateConfig   `yaml:"update_config,omitempty"`
	RollbackConfig *RollbackConfig `yaml:"rollback_config,omitempty"`
	RestartPolicy  *RestartPolicy  `yaml:"restart_policy,omitempty"`
	Resources      *Resources      `yaml:"resources,omitempty"`
}

// Placement represents placement constraints
type Placement struct {
	Constraints []string `yaml:"constraints,omitempty"`
	Preferences []string `yaml:"preferences,omitempty"`
}

// UpdateConfig represents update configuration
type UpdateConfig struct {
	Parallelism     int    `yaml:"parallelism,omitempty"`
	Delay           string `yaml:"delay,omitempty"`
	Order           string `yaml:"order,omitempty"`
	FailureAction   string `yaml:"failure_action,omitempty"`
	Monitor         string `yaml:"monitor,omitempty"`
	MaxFailureRatio string `yaml:"max_failure_ratio,omitempty"`
}

// RollbackConfig represents rollback configuration
type RollbackConfig struct {
	Parallelism     int    `yaml:"parallelism,omitempty"`
	Delay           string `yaml:"delay,omitempty"`
	Order           string `yaml:"order,omitempty"`
	FailureAction   string `yaml:"failure_action,omitempty"`
	Monitor         string `yaml:"monitor,omitempty"`
	MaxFailureRatio string `yaml:"max_failure_ratio,omitempty"`
}

// RestartPolicy represents restart policy configuration
type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty"`
	MaxAttempts int    `yaml:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty"`
}

// Resources represents resource limits and reservations
type Resources struct {
	Limits       *ResourceLimit `yaml:"limits,omitempty"`
	Reservations *ResourceLimit `yaml:"reservations,omitempty"`
}

// ResourceLimit represents resource limits
type ResourceLimit struct {
	CPUs   string `yaml:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty"`
}

// Ulimit represents ulimit configuration
type Ulimit struct {
	Soft int `yaml:"soft"`
	Hard int `yaml:"hard"`
}

// ModifyStack modifies a stack configuration based on the provided modifications
func ModifyStack(stack *Stack, modifications map[string]interface{}) (*Stack, error) {
	// Create a deep copy of the stack
	modifiedStack := &Stack{
		Services: make(map[string]Service),
	}

	// Copy existing services
	for name, service := range stack.Services {
		modifiedStack.Services[name] = service
	}

	// Apply modifications
	for path, value := range modifications {
		parts := strings.Split(path, ".")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid modification path: %s", path)
		}

		if parts[0] != "services" {
			return nil, fmt.Errorf("unsupported modification path: %s", path)
		}

		serviceName := parts[1]
		service, exists := modifiedStack.Services[serviceName]
		if !exists {
			return nil, fmt.Errorf("service not found: %s", serviceName)
		}

		field := parts[2]
		switch field {
		case "image":
			if strValue, ok := value.(string); ok {
				service.Image = strValue
			} else {
				return nil, fmt.Errorf("invalid image value type: %T", value)
			}
		case "ports":
			if ports, ok := value.([]Port); ok {
				service.Ports = ports
			} else {
				return nil, fmt.Errorf("invalid ports value type: %T", value)
			}
		case "environment":
			if env, ok := value.([]string); ok {
				service.Environment = env
			} else {
				return nil, fmt.Errorf("invalid environment value type: %T", value)
			}
		case "volumes":
			if volumes, ok := value.([]Volume); ok {
				service.Volumes = volumes
			} else {
				return nil, fmt.Errorf("invalid volumes value type: %T", value)
			}
		default:
			return nil, fmt.Errorf("unsupported field: %s", field)
		}

		modifiedStack.Services[serviceName] = service
	}

	return modifiedStack, nil
}

// ApplyCriteria applies the given criteria to the stack configuration
func ApplyCriteria(stack *Stack, criteria map[string]interface{}) (*Stack, error) {
	// Create a deep copy of the stack
	modifiedStack := &Stack{
		Services: make(map[string]Service),
	}

	// Copy existing services
	for name, service := range stack.Services {
		modifiedStack.Services[name] = service
	}

	// Apply criteria
	for path, value := range criteria {
		parts := strings.Split(path, ".")
		if len(parts) < 3 {
			return nil, fmt.Errorf("invalid criteria path: %s", path)
		}

		if parts[0] != "services" {
			return nil, fmt.Errorf("unsupported criteria path: %s", path)
		}

		serviceName := parts[1]
		service, exists := modifiedStack.Services[serviceName]
		if !exists {
			return nil, fmt.Errorf("service not found: %s", serviceName)
		}

		field := parts[2]
		switch field {
		case "image":
			if strValue, ok := value.(string); ok {
				service.Image = strValue
			} else {
				return nil, fmt.Errorf("invalid image value type: %T", value)
			}
		case "ports":
			if ports, ok := value.([]Port); ok {
				service.Ports = ports
			} else {
				return nil, fmt.Errorf("invalid ports value type: %T", value)
			}
		case "environment":
			if env, ok := value.([]string); ok {
				service.Environment = env
			} else {
				return nil, fmt.Errorf("invalid environment value type: %T", value)
			}
		case "volumes":
			if volumes, ok := value.([]Volume); ok {
				service.Volumes = volumes
			} else {
				return nil, fmt.Errorf("invalid volumes value type: %T", value)
			}
		default:
			return nil, fmt.Errorf("unsupported field: %s", field)
		}

		modifiedStack.Services[serviceName] = service
	}

	return modifiedStack, nil
}
