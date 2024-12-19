package aws

import (
	"context"
	"fmt"
	"time"
	"math/rand"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"log"
)

// Client wraps AWS SDK clients
type Client struct {
	ctx context.Context
	EC2 *ec2.Client
	SSM *ssm.Client
	region string
	verbose bool
}

// NewClient creates a new AWS client
func NewClient(region string, verbose bool) (*Client, error) {
	ctx := context.Background()

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create service clients
	return &Client{
		ctx: ctx,
		EC2: ec2.NewFromConfig(cfg),
		SSM: ssm.NewFromConfig(cfg),
		region: region,
		verbose: verbose,
	}, nil
}

// GetRegion returns the configured AWS region
func (c *Client) GetRegion() string {
	return c.region
}

// GetSSMEndpoint returns the SSM endpoint for the configured region
func (c *Client) GetSSMEndpoint() string {
	return fmt.Sprintf("https://ssm.%s.amazonaws.com", c.region)
}

// GetParameterValue gets a parameter value from SSM Parameter Store
func (c *Client) GetParameterValue(name string, withDecryption bool) (string, error) {
	// Create a context with timeout for the parameter fetch
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	// Get the parameter
	input := &ssm.GetParameterInput{
		Name:           aws.String(name),
		WithDecryption: aws.Bool(withDecryption),
	}

	result, err := c.SSM.GetParameter(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get parameter %s: %w", name, err)
	}

	if result.Parameter.Value == nil {
		return "", fmt.Errorf("parameter %s has no value", name)
	}

	return *result.Parameter.Value, nil
}

// GetParameter gets a parameter from SSM (alias for GetParameterValue for interface compatibility)
func (c *Client) GetParameter(name string) (string, error) {
	return c.GetParameterValue(name, true)
}

// GetParametersByPath gets parameters by their exact names
func (c *Client) GetParametersByPath(paths []string) ([]ssmtypes.Parameter, error) {
	// Create a context with timeout for the parameter fetch
	ctx, cancel := context.WithTimeout(c.ctx, 30*time.Second)
	defer cancel()

	var parameters []ssmtypes.Parameter
	batchSize := 10 // AWS allows up to 10 parameters per batch

	// Process parameters in batches
	for i := 0; i < len(paths); i += batchSize {
		end := i + batchSize
		if end > len(paths) {
			end = len(paths)
		}
		batch := paths[i:end]

		if c.verbose {
			log.Printf("Fetching batch of %d parameters", len(batch))
		}

		input := &ssm.GetParametersInput{
			Names:          batch,
			WithDecryption: aws.Bool(true),
		}

		output, err := c.SSM.GetParameters(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to get parameters: %w", err)
		}

		if c.verbose {
			log.Printf("Found %d parameters in batch", len(output.Parameters))
			if len(output.InvalidParameters) > 0 {
				log.Printf("Invalid parameters: %v", output.InvalidParameters)
			}
		}

		parameters = append(parameters, output.Parameters...)
	}

	if len(parameters) == 0 {
		return nil, fmt.Errorf("no parameters found")
	}

	if c.verbose {
		log.Printf("Found total of %d parameters", len(parameters))
	}

	return parameters, nil
}

// GetJumphost returns a random EC2 instance that matches the filter pattern
func (c *Client) GetJumphost(env string, filter string) (*types.Instance, error) {
	// Replace environment placeholder in filter
	filter = strings.ReplaceAll(filter, "${PLACEHOLDER}", env)

	// Create EC2 filter for the Name tag
	filters := []types.Filter{
		{
			Name:   aws.String("tag:Name"),
			Values: []string{filter},
		},
		{
			Name:   aws.String("instance-state-name"),
			Values: []string{"running"},
		},
	}

	// Describe instances with filter
	input := &ec2.DescribeInstancesInput{
		Filters: filters,
	}

	output, err := c.EC2.DescribeInstances(context.TODO(), input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	// Collect all instances
	var instances []types.Instance
	for _, reservation := range output.Reservations {
		for _, instance := range reservation.Instances {
			if instance.State != nil && instance.State.Name == types.InstanceStateNameRunning {
				instances = append(instances, instance)
			}
		}
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("no running instances found matching filter: %s", filter)
	}

	// Return a random instance
	rand.Seed(time.Now().UnixNano())
	return &instances[rand.Intn(len(instances))], nil
}

// Helper function to get instance name from tags
func getInstanceName(instance *types.Instance) string {
	for _, tag := range instance.Tags {
		if *tag.Key == "Name" {
			return *tag.Value
		}
	}
	return ""
}
