package tunnel

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"

	awsclient "tunnel-go/pkg/aws"
	"tunnel-go/pkg/config"
)

// Manager handles tunnel creation and management
type Manager struct {
	client   *awsclient.Client
	config   *config.Config
	env      string
	tunnels  sync.Map
	verbose  bool
	jumphost *types.Instance
}

// NewManager creates a new tunnel manager
func NewManager(client *awsclient.Client, cfg *config.Config, env string, verbose bool) *Manager {
	return &Manager{
		client:  client,
		config:  cfg,
		env:     env,
		verbose: verbose,
	}
}

// CreateTunnel creates an SSM port forwarding tunnel for a service
func (m *Manager) CreateTunnel(serviceName string, serviceConfig config.ServiceConfig) error {
	if m.verbose {
		log.Printf("Creating tunnel for service: %s", serviceName)
	}

	// Get host and port
	host, err := serviceConfig.Host.GetValue(m.client, m.env)
	if err != nil {
		return fmt.Errorf("failed to get host for %s: %w", serviceName, err)
	}
	if m.verbose {
		log.Printf("Retrieved host for %s: %s", serviceName, host)
	}

	remotePort, err := serviceConfig.RemotePort.GetValue(m.client, m.env)
	if err != nil {
		return fmt.Errorf("failed to get remote port for %s: %w", serviceName, err)
	}
	if m.verbose {
		log.Printf("Retrieved remote port for %s: %s", serviceName, remotePort)
	}

	// Get jumphost instance if not already set
	if m.jumphost == nil {
		instance, err := m.GetJumphost()
		if err != nil {
			return fmt.Errorf("failed to find jumphost instance: %w", err)
		}
		m.jumphost = instance
		if m.verbose {
			log.Printf("Using jumphost instance: %s", *instance.InstanceId)
		}
	}

	// Find an available local port in the configured range
	localPort, err := findAvailablePort(serviceConfig.LocalPortRange.Start, serviceConfig.LocalPortRange.End)
	if err != nil {
		return fmt.Errorf("failed to find available port for %s: %w", serviceName, err)
	}
	if m.verbose {
		log.Printf("Found available local port for %s: %d", serviceName, localPort)
	}

	// Use AWS CLI to create the tunnel
	args := []string{
		"ssm",
		"start-session",
		"--target", *m.jumphost.InstanceId,
		"--document-name", "AWS-StartPortForwardingSessionToRemoteHost",
		"--parameters", fmt.Sprintf(`{"host":["%s"],"portNumber":["%s"],"localPortNumber":["%d"]}`, host, remotePort, localPort),
	}

	if m.verbose {
		log.Printf("Starting AWS CLI command: aws %s", strings.Join(args, " "))
	}

	cmd := exec.Command("aws", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run the command in the background
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start AWS CLI command: %w", err)
	}

	if m.verbose {
		log.Printf("AWS CLI command started successfully with PID: %d", cmd.Process.Pid)
	}

	// Store the command for cleanup
	m.tunnels.Store(serviceName, cmd)

	log.Printf("Created tunnel for %s: localhost:%d -> %s:%s", serviceName, localPort, host, remotePort)
	return nil
}

// CreateTunnels creates tunnels for multiple services
func (m *Manager) CreateTunnels(services []string) error {
	// Get jumphost instance first
	instance, err := m.GetJumphost()
	if err != nil {
		return fmt.Errorf("failed to find jumphost instance: %w", err)
	}
	m.jumphost = instance

	// Log jumphost information
	instanceName := getInstanceName(instance)
	log.Printf("Using jumphost: %s (%s)", instanceName, *instance.InstanceId)

	var lastError error
	// Create tunnels for each service
	for _, serviceName := range services {
		serviceConfig, err := m.config.GetServiceConfig(serviceName)
		if err != nil {
			log.Printf("Error getting config for %s: %v", serviceName, err)
			lastError = err
			continue
		}

		if err := m.CreateTunnel(serviceName, serviceConfig); err != nil {
			log.Printf("Error creating tunnel for %s: %v", serviceName, err)
			lastError = err
			continue
		}
	}
	
	if lastError != nil {
		return fmt.Errorf("one or more tunnels failed to create: %w", lastError)
	}
	return nil
}

// GetJumphost returns the EC2 instance to be used as a jumphost
func (m *Manager) GetJumphost() (*types.Instance, error) {
	filter := m.config.GetJumphostFilter(m.env)
	if m.verbose {
		log.Printf("Looking for jumphost with filter: %s", filter)
	}
	return m.client.GetJumphost(m.env, filter)
}

// GetServiceDetails retrieves SSM parameter values for a service
func (m *Manager) GetServiceDetails(serviceName string, serviceConfig config.ServiceConfig) (map[string]string, error) {
	details := make(map[string]string)

	// Get host parameter
	host, err := serviceConfig.Host.GetValue(m.client, m.env)
	if err != nil {
		return nil, fmt.Errorf("failed to get host: %w", err)
	}
	details["host"] = host

	// Get remote port parameter
	remotePort, err := serviceConfig.RemotePort.GetValue(m.client, m.env)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote port: %w", err)
	}
	details["remote_port"] = remotePort

	// Get service-specific details if configured
	if len(serviceConfig.ServiceDetails) > 0 {
		// Replace placeholder in all paths
		var paramPaths []string
		for _, path := range serviceConfig.ServiceDetails {
			paramPath := strings.ReplaceAll(path, "${PLACEHOLDER}", m.env)
			paramPaths = append(paramPaths, paramPath)
		}

		if m.verbose {
			log.Printf("Getting parameters: %v", paramPaths)
		}

		// Get all parameters
		parameters, err := m.client.GetParametersByPath(paramPaths)
		if err != nil {
			log.Printf("Warning: Failed to get parameters: %v", err)
			return details, nil
		}

		// Add all parameters to the details map
		for _, param := range parameters {
			if param.Name == nil || param.Value == nil {
				continue
			}
			name := *param.Name
			// Extract just the parameter name without the path
			parts := strings.Split(name, "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
			details[name] = *param.Value
		}
	}

	// Add local port range for reference
	details["local_port_range"] = fmt.Sprintf("%d-%d", 
		serviceConfig.LocalPortRange.Start, 
		serviceConfig.LocalPortRange.End)

	return details, nil
}

// Helper function to get instance name from tags
func getInstanceName(instance *types.Instance) string {
	for _, tag := range instance.Tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return "unnamed"
}

// isPortAvailable checks if a port is available for use
func isPortAvailable(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	ln.Close()
	return true
}

// findAvailablePort finds an available port in the given range
func findAvailablePort(start, end int) (int, error) {
	for port := start; port <= end; port++ {
		if isPortAvailable(port) {
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports in range %d-%d", start, end)
}

// CleanupTunnels terminates all active tunnels
func (m *Manager) CleanupTunnels() error {
	var lastErr error
	m.tunnels.Range(func(key, value interface{}) bool {
		cmd := value.(*exec.Cmd)
		if err := cmd.Process.Kill(); err != nil {
			lastErr = fmt.Errorf("failed to kill tunnel process: %w", err)
			return false
		}
		return true
	})
	return lastErr
}
