package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ConfigValue represents a value that can be either a direct value or an SSM parameter
type ConfigValue struct {
	Value    string `yaml:"value"`
	SSMParam string `yaml:"ssm_param"`
}

// PortRange represents a range of ports
type PortRange struct {
	Start int `yaml:"start"`
	End   int `yaml:"end"`
}

// ServiceConfig represents the configuration for a service
type ServiceConfig struct {
	Host           ConfigValue `yaml:"host"`
	RemotePort     ConfigValue `yaml:"remote-port"`
	LocalPortRange PortRange   `yaml:"local-port-range"`
	ServiceDetails []string    `yaml:"service-details,omitempty"`
}

// Config represents the configuration file structure
type Config struct {
	DefaultRegion string `yaml:"default_region"`
	AWS           struct {
		Profile string `yaml:"profile"`
	} `yaml:"aws"`
	TunnelConfig struct {
		Placeholder       string                   `yaml:"placeholder"`
		CachefileLocation string                   `yaml:"cachefile-location"`
		LogfileLocation   string                   `yaml:"logfile-location"`
		JumphostFilter    string                   `yaml:"jumphost-filter"`
		Services          map[string]ServiceConfig `yaml:"services"`
	} `yaml:"tunnel-go-config"`
}

// SSMClient interface for AWS SSM operations
type SSMClient interface {
	GetParameter(name string) (string, error)
}

// GetValue returns either the direct value or fetches from SSM if SSMParam is set
func (cv *ConfigValue) GetValue(ssmClient SSMClient, placeholder string) (string, error) {
	// Check if both value and SSM parameter are specified
	if cv.SSMParam != "" && cv.Value != "" {
		return "", fmt.Errorf("cannot specify both value and SSM parameter")
	}

	if cv.SSMParam != "" {
		// Replace placeholder in SSM parameter path
		paramPath := strings.ReplaceAll(cv.SSMParam, "${PLACEHOLDER}", placeholder)
		return ssmClient.GetParameter(paramPath)
	}
	if cv.Value != "" {
		return cv.Value, nil
	}
	return "", fmt.Errorf("no value or SSM parameter specified")
}

// GetJumphostFilter returns the jumphost filter pattern for the given environment
func (c *Config) GetJumphostFilter(env string) string {
	return strings.ReplaceAll(c.TunnelConfig.JumphostFilter, "${PLACEHOLDER}", env)
}

// GetServiceConfig returns the configuration for a specific service
func (c *Config) GetServiceConfig(serviceName string) (ServiceConfig, error) {
	if service, ok := c.TunnelConfig.Services[serviceName]; ok {
		return service, nil
	}
	return ServiceConfig{}, fmt.Errorf("unknown service: %s", serviceName)
}

// LoadConfig reads the configuration from a YAML file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %w", err)
	}

	return &config, nil
}
