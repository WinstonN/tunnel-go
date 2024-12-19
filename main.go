package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sort"
	"strings"

	"tunnel-go/pkg/aws"
	"tunnel-go/pkg/config"
	"tunnel-go/pkg/tunnel"
)

const helpText = `
AWS SSM Tunnel Tool - Create secure tunnels to AWS resources

Usage:
  tunnel-go [command] [flags]

Commands:
  create-tunnel    Create SSH tunnels to specified services
  service-details  Query SSM parameters for specified services

Flags:
  -config string
        Path to config file (default: searches in ./config.yaml, ~/.tunnel/config.yaml)
  -env string
        Environment to use (e.g., dev, staging, prod)
  -services string
        Comma-separated list of services to tunnel to (e.g., "database,redis")
  -region string
        AWS region (overrides config file)
  -verbose
        Enable verbose logging

Examples:
  # Create tunnels for database and redis in production
  tunnel-go create-tunnel -env prod -services "database,redis"

  # Query service details from SSM
  tunnel-go service-details -env prod -services "database"

  # Use a specific config file
  tunnel-go create-tunnel -config /path/to/config.yaml -services "database"

  # Create tunnels in a specific region
  tunnel-go create-tunnel -region us-west-2 -env staging -services "database"

Configuration:
  The config file should be in YAML format with the following structure:
  
  aws:
    default_region: us-west-2
  tunnel-go-config:
    placeholder: environment
    jumphost-filter: "${PLACEHOLDER}-autoscaled"
    services:
      database:
        host:
          ssm_param: "/${PLACEHOLDER}/path/to/parameter/DB_HOST"
          value: ""
        remote-port:
          ssm_param: "/${PLACEHOLDER}/path/to/parameter/DB_PORT"
          value: ""
        local-port-range:
          start: 5000
          end: 5009
        service-details:
          - /${PLACEHOLDER}/path/to/parameter/DB_HOST

For more information, visit: https://github.com/WinstonN/tunnel-go
`

func findConfigFile(configPath string) (string, error) {
	// If config path is provided, use it
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		return "", fmt.Errorf("config file not found at %s", configPath)
	}

	// Check common locations
	locations := []string{
		"config.yaml",
		"tunnel-go.yaml",
		"~/.tunnel-go/config.yaml",
		"~/.config/tunnel-go.yaml",
	}

	for _, loc := range locations {
		expanded := strings.Replace(loc, "~", os.Getenv("HOME"), 1)
		if _, err := os.Stat(expanded); err == nil {
			return expanded, nil
		}
	}

	return "", fmt.Errorf("no config file found in standard locations")
}

func main() {
	// Define flags
	createTunnelCmd := flag.NewFlagSet("create-tunnel", flag.ExitOnError)
	createTunnelConfig := createTunnelCmd.String("config", "", "Path to config file")
	createTunnelEnv := createTunnelCmd.String("env", "", "Environment name")
	createTunnelServices := createTunnelCmd.String("services", "", "Comma-separated list of services")
	createTunnelRegion := createTunnelCmd.String("region", "", "AWS region (optional, overrides config default_region)")
	createTunnelVerbose := createTunnelCmd.Bool("verbose", false, "Enable verbose logging")

	serviceDetailsCmd := flag.NewFlagSet("service-details", flag.ExitOnError)
	serviceDetailsConfig := serviceDetailsCmd.String("config", "", "Path to config file")
	serviceDetailsEnv := serviceDetailsCmd.String("env", "", "Environment name")
	serviceDetailsServices := serviceDetailsCmd.String("services", "", "Comma-separated list of services")
	serviceDetailsRegion := serviceDetailsCmd.String("region", "", "AWS region (optional, overrides config default_region)")
	serviceDetailsVerbose := serviceDetailsCmd.Bool("verbose", false, "Enable verbose logging")

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println(helpText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-tunnel":
		err := createTunnelCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalf("Failed to parse flags: %v", err)
		}

		if *createTunnelEnv == "" {
			log.Fatal("Environment name is required")
		}
		if *createTunnelServices == "" {
			log.Fatal("Services list is required")
		}

		// Find config file
		foundConfigPath, err := findConfigFile(*createTunnelConfig)
		if err != nil {
			log.Fatalf("Failed to find config file: %v", err)
		}

		// Load the configuration
		cfg, err := config.LoadConfig(foundConfigPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		// Use region from flag if provided, otherwise use default from config
		region := cfg.DefaultRegion
		if *createTunnelRegion != "" {
			region = *createTunnelRegion
		}

		// Initialize AWS client
		awsClient, err := aws.NewClient(region, *createTunnelVerbose)
		if err != nil {
			log.Fatalf("Failed to create AWS client: %v", err)
		}

		// Create tunnel manager
		manager := tunnel.NewManager(awsClient, cfg, *createTunnelEnv, *createTunnelVerbose)

		// Parse services list
		services := strings.Split(*createTunnelServices, ",")

		// Create tunnels
		if err := manager.CreateTunnels(services); err != nil {
			log.Fatalf("Failed to create tunnels: %v", err)
		}

		fmt.Print("Tunnels created successfully. Press Ctrl+C to exit and close all tunnels")

		// Wait for interrupt signal
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt)
		<-sigChan

	case "service-details":
		err := serviceDetailsCmd.Parse(os.Args[2:])
		if err != nil {
			log.Fatalf("Failed to parse flags: %v", err)
		}

		if *serviceDetailsEnv == "" {
			log.Fatal("Environment name is required")
		}
		if *serviceDetailsServices == "" {
			log.Fatal("Services list is required")
		}

		// Find config file
		foundConfigPath, err := findConfigFile(*serviceDetailsConfig)
		if err != nil {
			log.Fatalf("Failed to find config file: %v", err)
		}

		// Load the configuration
		cfg, err := config.LoadConfig(foundConfigPath)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}

		// Use region from flag if provided, otherwise use default from config
		region := cfg.DefaultRegion
		if *serviceDetailsRegion != "" {
			region = *serviceDetailsRegion
		}

		// Initialize AWS client
		awsClient, err := aws.NewClient(region, *serviceDetailsVerbose)
		if err != nil {
			log.Fatalf("Failed to create AWS client: %v", err)
		}

		// Create tunnel manager
		manager := tunnel.NewManager(awsClient, cfg, *serviceDetailsEnv, *serviceDetailsVerbose)

		// Get details for each service
		services := strings.Split(*serviceDetailsServices, ",")
		for _, serviceName := range services {
			serviceConfig, err := cfg.GetServiceConfig(serviceName)
			if err != nil {
				log.Fatalf("Failed to get config for %s: %v", serviceName, err)
			}

			details, err := manager.GetServiceDetails(serviceName, serviceConfig)
			if err != nil {
				log.Printf("Warning: Failed to get details for %s: %v", serviceName, err)
				continue
			}

			fmt.Printf("\nService: %s\n", serviceName)

			// Sort and print additional parameters
			var keys []string
			for k := range details {
				if k != "host" && k != "remote_port" && k != "local_port_range" {
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)

			if len(keys) > 0 {
				for _, k := range keys {
					fmt.Printf("%s=%s\n", k, details[k])
				}
			}
			fmt.Println()
		}
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
