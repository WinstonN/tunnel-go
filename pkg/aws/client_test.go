package aws

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type mockSSMClient struct {
	getParameterOutput *ssm.GetParameterOutput
	err               error
}

func (m *mockSSMClient) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.getParameterOutput, nil
}

type mockEC2Client struct {
	describeInstancesOutput *ec2.DescribeInstancesOutput
	err                    error
}

func (m *mockEC2Client) DescribeInstances(ctx context.Context, params *ec2.DescribeInstancesInput, optFns ...func(*ec2.Options)) (*ec2.DescribeInstancesOutput, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.describeInstancesOutput, nil
}

func TestGetEC2InstancesByFilter(t *testing.T) {
	tests := []struct {
		name    string
		filter  string
		output  *ec2.DescribeInstancesOutput
		err     error
		want    []string
		wantErr bool
	}{
		{
			name:   "Success",
			filter: "test-filter",
			output: &ec2.DescribeInstancesOutput{
				Reservations: []ec2types.Reservation{
					{
						Instances: []ec2types.Instance{
							{
								PrivateIpAddress: aws.String("10.0.0.1"),
							},
							{
								PrivateIpAddress: aws.String("10.0.0.2"),
							},
						},
					},
				},
			},
			want:    []string{"10.0.0.1", "10.0.0.2"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEC2 := &mockEC2Client{
				describeInstancesOutput: tt.output,
				err:                    tt.err,
			}

			client := NewClient(nil, mockEC2)

			got, err := client.GetEC2InstancesByFilter(tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEC2InstancesByFilter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("GetEC2InstancesByFilter() got %v, want %v", got, tt.want)
				}
				for i, ip := range got {
					if ip != tt.want[i] {
						t.Errorf("GetEC2InstancesByFilter() got[%d] = %v, want %v", i, ip, tt.want[i])
					}
				}
			}
		})
	}
}

func TestGetParameter(t *testing.T) {
	tests := []struct {
		name    string
		param   string
		output  *ssm.GetParameterOutput
		err     error
		want    string
		wantErr bool
	}{
		{
			name:  "Success",
			param: "/test/param",
			output: &ssm.GetParameterOutput{
				Parameter: &ssmtypes.Parameter{
					Value: aws.String("test-value"),
				},
			},
			want:    "test-value",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSSM := &mockSSMClient{
				getParameterOutput: tt.output,
				err:               tt.err,
			}

			client := NewClient(mockSSM, nil)

			got, err := client.GetParameter(tt.param)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}
