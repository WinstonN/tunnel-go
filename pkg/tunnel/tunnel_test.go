package tunnel

import (
	"errors"
	"testing"

	"tunnel-go/pkg/aws"
	"tunnel-go/pkg/config"
)

type mockAWSClient struct {
	getInstancesOutput []string
	getParamOutput     string
	err                error
}

func (m *mockAWSClient) GetEC2InstancesByFilter(filter string) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.getInstancesOutput, nil
}

func (m *mockAWSClient) GetParameter(name string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.getParamOutput, nil
}

func TestGetJumphosts(t *testing.T) {
	tests := []struct {
		name       string
		filter     string
		mockOutput []string
		mockErr    error
		want       []string
		wantErr    bool
	}{
		{
			name:       "Success",
			filter:     "test-filter",
			mockOutput: []string{"10.0.0.1", "10.0.0.2"},
			want:       []string{"10.0.0.1", "10.0.0.2"},
			wantErr:    false,
		},
		{
			name:    "Error",
			filter:  "test-filter",
			mockErr: errors.New("test error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAWSClient{
				getInstancesOutput: tt.mockOutput,
				err:                tt.mockErr,
			}

			manager := &Manager{
				client: aws.NewClient(mock, mock),
				config: &config.Config{
					TunnelConfig: struct {
						Placeholder       string                          `yaml:"placeholder"`
						CachefileLocation string                          `yaml:"cachefile-location"`
						LogfileLocation   string                          `yaml:"logfile-location"`
						JumphostFilter    string                          `yaml:"jumphost-filter"`
						Services          map[string]config.ServiceConfig `yaml:"services"`
					}{
						JumphostFilter: tt.filter,
					},
				},
				env:     "test",
				verbose: true,
			}

			got, err := manager.GetJumphosts()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetJumphosts() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("GetJumphosts() returned %d jumphosts, want %d", len(got), len(tt.want))
					return
				}
				for i, host := range got {
					if host != tt.want[i] {
						t.Errorf("GetJumphosts()[%d] = %v, want %v", i, host, tt.want[i])
					}
				}
			}
		})
	}
}

func TestCreateTunnels(t *testing.T) {
	tests := []struct {
		name       string
		services   []string
		mockOutput string
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "Success",
			services:   []string{"service1"},
			mockOutput: "test-value",
			wantErr:    false,
		},
		{
			name:     "Error",
			services: []string{"service1"},
			mockErr:  errors.New("test error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockAWSClient{
				getParamOutput: tt.mockOutput,
				err:            tt.mockErr,
			}

			manager := &Manager{
				client: aws.NewClient(mock, mock),
				config: &config.Config{
					TunnelConfig: struct {
						Placeholder       string                          `yaml:"placeholder"`
						CachefileLocation string                          `yaml:"cachefile-location"`
						LogfileLocation   string                          `yaml:"logfile-location"`
						JumphostFilter    string                          `yaml:"jumphost-filter"`
						Services          map[string]config.ServiceConfig `yaml:"services"`
					}{
						Services: map[string]config.ServiceConfig{
							"service1": {
								Host: config.ConfigValue{
									Value: "test-host",
								},
								RemotePort: config.ConfigValue{
									Value: "3306",
								},
								LocalPortRange: config.PortRange{
									Start: 5000,
									End:   5009,
								},
							},
						},
					},
				},
				env:     "test",
				verbose: true,
			}

			err := manager.CreateTunnels(tt.services)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTunnels() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
