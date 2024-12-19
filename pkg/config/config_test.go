package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Mock implementation
type mockSSMClient struct {
	params map[string]string
	err    error
}

func (m *mockSSMClient) GetParameter(name string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	if val, ok := m.params[name]; ok {
		return val, nil
	}
	return "", fmt.Errorf("parameter not found: %s", name)
}

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	configContent := `default_region: eu-central-1
aws:
  profile: default
tunnel-go-config:
  placeholder: environment
  jumphost-filter: ${PLACEHOLDER}-ecs-autoscaled
  cachefile-location: /tmp/tunnel-cache
  logfile-location: /tmp/tunnel.log
  services:
    database:
      host:
        ssm_param: "/${PLACEHOLDER}/service/database/host"
        value: ""
      remote-port:
        ssm_param: "/${PLACEHOLDER}/service/database/port"
        value: ""
      local-port-range:
        start: 5000
        end: 5009
      service-details:
        - /${PLACEHOLDER}/service/database/host`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Test loading config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify config values
	if cfg.DefaultRegion != "eu-central-1" {
		t.Errorf("Expected default region eu-central-1, got %s", cfg.DefaultRegion)
	}

	if cfg.AWS.Profile != "default" {
		t.Errorf("Expected AWS profile 'default', got %s", cfg.AWS.Profile)
	}

	if cfg.TunnelConfig.Placeholder != "environment" {
		t.Errorf("Expected placeholder 'environment', got %s", cfg.TunnelConfig.Placeholder)
	}

	// Test service config
	dbConfig, err := cfg.GetServiceConfig("database")
	if err != nil {
		t.Fatalf("GetServiceConfig failed: %v", err)
	}

	if dbConfig.Host.SSMParam != "/${PLACEHOLDER}/service/database/host" {
		t.Errorf("Expected host SSM param /${PLACEHOLDER}/service/database/host, got %s", dbConfig.Host.SSMParam)
	}

	if dbConfig.LocalPortRange.Start != 5000 {
		t.Errorf("Expected port range start 5000, got %d", dbConfig.LocalPortRange.Start)
	}

	if dbConfig.LocalPortRange.End != 5009 {
		t.Errorf("Expected port range end 5009, got %d", dbConfig.LocalPortRange.End)
	}
}

func TestGetValue(t *testing.T) {
	tests := []struct {
		name       string
		config     ConfigValue
		ssmClient  *mockSSMClient
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name: "Direct value",
			config: ConfigValue{
				Value: "test-value",
			},
			want:    "test-value",
			wantErr: false,
		},
		{
			name: "SSM parameter",
			config: ConfigValue{
				SSMParam: "/test/param",
			},
			ssmClient: &mockSSMClient{
				params: map[string]string{
					"/test/param": "ssm-value",
				},
			},
			want:    "ssm-value",
			wantErr: false,
		},
		{
			name: "Both value and SSM parameter specified",
			config: ConfigValue{
				Value:    "test-value",
				SSMParam: "/test/param",
			},
			wantErr:    true,
			errMessage: "cannot specify both value and SSM parameter",
		},
		{
			name:       "No value or SSM parameter",
			config:     ConfigValue{},
			wantErr:    true,
			errMessage: "no value or SSM parameter specified",
		},
		{
			name: "SSM parameter with placeholder",
			config: ConfigValue{
				SSMParam: "/${PLACEHOLDER}/param",
			},
			ssmClient: &mockSSMClient{
				params: map[string]string{
					"/test/param": "ssm-value",
				},
			},
			want:    "ssm-value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ssmClient == nil {
				tt.ssmClient = &mockSSMClient{}
			}

			got, err := tt.config.GetValue(tt.ssmClient, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMessage != "" && err.Error() != tt.errMessage {
				t.Errorf("GetValue() error message = %v, want %v", err.Error(), tt.errMessage)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetValueValidation(t *testing.T) {
	tests := []struct {
		name       string
		config     ConfigValue
		ssmClient  *mockSSMClient
		want       string
		wantErr    bool
		errMessage string
	}{
		{
			name: "Direct value",
			config: ConfigValue{
				Value: "test-value",
			},
			want:    "test-value",
			wantErr: false,
		},
		{
			name: "SSM parameter",
			config: ConfigValue{
				SSMParam: "/test/param",
			},
			ssmClient: &mockSSMClient{
				params: map[string]string{
					"/test/param": "ssm-value",
				},
			},
			want:    "ssm-value",
			wantErr: false,
		},
		{
			name: "Both value and SSM parameter specified",
			config: ConfigValue{
				Value:    "test-value",
				SSMParam: "/test/param",
			},
			wantErr:    true,
			errMessage: "cannot specify both value and SSM parameter",
		},
		{
			name:       "No value or SSM parameter",
			config:     ConfigValue{},
			wantErr:    true,
			errMessage: "no value or SSM parameter specified",
		},
		{
			name: "SSM parameter with placeholder",
			config: ConfigValue{
				SSMParam: "/${PLACEHOLDER}/param",
			},
			ssmClient: &mockSSMClient{
				params: map[string]string{
					"/test/param": "ssm-value",
				},
			},
			want:    "ssm-value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.ssmClient == nil {
				tt.ssmClient = &mockSSMClient{}
			}

			got, err := tt.config.GetValue(tt.ssmClient, "test")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMessage != "" && err.Error() != tt.errMessage {
				t.Errorf("GetValue() error message = %v, want %v", err.Error(), tt.errMessage)
				return
			}
			if got != tt.want {
				t.Errorf("GetValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetServiceConfig(t *testing.T) {
	cfg := &Config{
		TunnelConfig: struct {
			Placeholder       string                   `yaml:"placeholder"`
			CachefileLocation string                   `yaml:"cachefile-location"`
			LogfileLocation   string                   `yaml:"logfile-location"`
			JumphostFilter    string                   `yaml:"jumphost-filter"`
			Services          map[string]ServiceConfig `yaml:"services"`
		}{
			Services: map[string]ServiceConfig{
				"database": {
					Host: ConfigValue{
						SSMParam: "/${PLACEHOLDER}/db/host",
					},
					RemotePort: ConfigValue{
						Value: "3306",
					},
					LocalPortRange: PortRange{
						Start: 5000,
						End:   5009,
					},
				},
			},
		},
	}

	tests := []struct {
		name        string
		serviceName string
		wantErr     bool
	}{
		{
			name:        "Existing service",
			serviceName: "database",
			wantErr:     false,
		},
		{
			name:        "Non-existent service",
			serviceName: "invalid",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := cfg.GetServiceConfig(tt.serviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetJumphostFilter(t *testing.T) {
	cfg := &Config{
		TunnelConfig: struct {
			Placeholder       string                   `yaml:"placeholder"`
			CachefileLocation string                   `yaml:"cachefile-location"`
			LogfileLocation   string                   `yaml:"logfile-location"`
			JumphostFilter    string                   `yaml:"jumphost-filter"`
			Services          map[string]ServiceConfig `yaml:"services"`
		}{
			Placeholder:     "environment",
			JumphostFilter: "${PLACEHOLDER}-ecs-autoscaled",
		},
	}

	tests := []struct {
		name string
		env  string
		want string
	}{
		{
			name: "Basic replacement",
			env:  "prod",
			want: "prod-ecs-autoscaled",
		},
		{
			name: "Empty environment",
			env:  "",
			want: "-ecs-autoscaled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetJumphostFilter(tt.env)
			if got != tt.want {
				t.Errorf("GetJumphostFilter() = %v, want %v", got, tt.want)
			}
		})
	}
}
