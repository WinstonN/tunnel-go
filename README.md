# TunnelGo

## Overview

TunnelGo is a CLI tool that allows you to create tunnels to remote hosts using AWS SSM. It is a simple and easy to use tool that can be used to create a tunnel to a remote host using AWS SSM.

## Prerequisites

- Go 1.21 or later (REQUIRED - the AWS SDK dependencies require Go 1.21)
- AWS credentials configured (either via environment variables, AWS CLI configuration, or IAM role)
- Access to AWS SSM and EC2 services

## Building

1. First, ensure you have Go 1.21 or later installed:
```bash
go version
```

If you have an older version, you must upgrade Go to 1.21 or later:

### macOS
```bash
# Using Homebrew
brew update
brew upgrade go
# Or install a specific version
brew install go@1.21
```

### Linux (Ubuntu/Debian)
```bash
# Add the Go repository
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt update
sudo apt install golang-1.21
```

### Manual Installation
Visit https://golang.org/dl/ to download and install Go 1.21 or later for your platform.

2. Clone the repository:
```bash
git clone <repository-url>
cd tunnel-go
```

3. Install dependencies:
```bash
go mod tidy
```

4. Build the binary:
```bash
go build -o tunnel-go
```

5. (Optional) Install the binary to your system:
```bash
sudo mv tunnel-go /usr/local/bin/
```

## Configuration

The tool looks for a configuration file in the following locations (in order):
1. Path specified by `--config` flag
2. `config.yaml` in the current directory
3. `tunnel-go.yaml` in the current directory
4. `~/.tunnel-go/config.yaml`
5. `~/.config/tunnel-go.yaml`

See `config-example.yaml` for an example configuration file.

## AWS Configuration

The tool supports multiple ways to configure AWS credentials and region, in the following order of precedence:

1. Environment Variables:
   - `AWS_REGION` or `AWS_DEFAULT_REGION`: AWS region
   - `AWS_PROFILE`: AWS credential profile name
   - `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`: Direct credentials
   - `AWS_SESSION_TOKEN`: For temporary credentials
   - `AWS_ROLE_ARN`: For role assumption

2. Configuration File:
   ```yaml
   aws:
     default_region: eu-central-1
     profile: my-profile  # optional
   ```

3. AWS CLI Configuration:
   - `~/.aws/config`
   - `~/.aws/credentials`

4. IAM Instance Profile (when running on EC2)

For security best practices, we recommend using environment variables or IAM roles instead of hardcoding credentials.

## Port Management

Each service in the configuration can specify a range of local ports to use for tunneling:

```yaml
services:
  database:
    local-port-range:
      start: 5000
      end: 5009
```

The tool will:
1. Start with the first port in the range (e.g., 5000)
2. Check if the port is available
3. If the port is busy:
   - Try the next port in the range
   - Continue until it finds an available port
   - Error if no ports are available in the range

This allows multiple instances of the tool to run simultaneously without port conflicts.

## Usage

The tool supports two main commands:

### Create Tunnels

Creates SSM port forwarding tunnels for the specified services:

```bash
tunnel-go create-tunnel -services "database,redis" -env dev
```

### Get Service Details

Retrieves and displays service details from SSM parameters:

```bash
tunnel-go service-details -services "database,redis" -env dev
```

### Command Line Options

- `--command`: Command to run (`create-tunnel` or `service-details`, defaults to `create-tunnel`)
- `-services`: Space-separated list of services to tunnel (required)
- `-env`: Environment name (required)
- `-config`: Path to configuration file (optional)

## Logging

The tool logs its output to:
- Standard output

## Development

To contribute to the project:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

[Add your license information here]
