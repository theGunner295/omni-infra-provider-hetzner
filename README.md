# omni-infra-provider-hetzner

A [Sidero Omni](https://github.com/siderolabs/omni) Infrastructure Provider for [Hetzner Cloud](https://www.hetzner.com/cloud), implemented in Go following the same architecture as the [vSphere provider](https://github.com/siderolabs/omni-infra-provider-vsphere).

## Features

- **Multi-project support**: Manage VMs across multiple Hetzner Cloud projects simultaneously, each with its own API token, snapshot, and default location.
- **VM provisioning from snapshots**: Create Hetzner servers from user-specified snapshot images (by name).
- **Flexible server types/SKUs**: Specify any Hetzner server type (e.g. `cx23`, `cx32`, `cx52`, `cpx21`).
- **Location/zone selection**: Deploy to any Hetzner location (`fsn1`, `nbg1`, `hel1`, etc.) with SKU+zone validation.
- **Dual networking modes**:
  - **Public IP mode** (default): allocates public IPv4+IPv6 automatically.
  - **Private VNet mode**: attaches to an existing Hetzner private network without a public IP.
- **Resilient API client**: Exponential backoff retry logic for rate limiting, transient errors, and network failures.
- **Full VM lifecycle**: create, power management, metadata injection (Talos join config via user-data), and clean deletion.

## Building

```bash
go build ./cmd/omni-infra-provider-hetzner/
```

## Configuration

Create a YAML configuration file:

```yaml
projects:
  - token: "your-hetzner-api-token"
    project_id: "my-project"
    snapshot_name: "talos-v1.9-amd64"
    default_location: "fsn1"

  # Optional: second project
  - token: "another-hetzner-api-token"
    project_id: "second-project"
    snapshot_name: "talos-v1.9-amd64"
    default_location: "nbg1"
```

### Configuration Fields

| Field | Required | Description |
|-------|----------|-------------|
| `projects[].token` | ✅ | Hetzner Cloud API token for this project |
| `projects[].project_id` | ✅ | Unique identifier for this project (used for per-machine selection) |
| `projects[].snapshot_name` | ✅ | Default snapshot image name for VM provisioning |
| `projects[].default_location` | ❌ | Default Hetzner location (e.g. `fsn1`, `nbg1`, `hel1`) |

## Running

```bash
omni-infra-provider-hetzner \
  --omni-api-endpoint=https://your-omni-instance \
  --omni-service-account-key=<key> \
  --config-file=/path/to/config.yaml \
  --provider-name=hetzner \
  --id=hetzner
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--omni-api-endpoint` | `$OMNI_ENDPOINT` | Omni API endpoint URL |
| `--omni-service-account-key` | `$OMNI_SERVICE_ACCOUNT_KEY` | Omni service account key |
| `--config-file` | _(required)_ | Path to provider configuration YAML |
| `--id` | `hetzner` | Provider ID (must match the label in Omni) |
| `--provider-name` | `hetzner` | Human-readable provider name in Omni |
| `--provider-description` | `Hetzner Cloud infrastructure provider` | Provider description in Omni |
| `--insecure-skip-verify` | `false` | Skip TLS verification for Omni connection |

## Machine Schema

Each machine request can include per-machine provider configuration:

```yaml
server_type: cx32        # Required: Hetzner server type/SKU
project_id: my-project   # Optional: selects project (required if multiple projects)
location: fsn1           # Optional: overrides project default_location
snapshot_name: talos-custom  # Optional: overrides project snapshot_name

# Networking
network_mode: public     # "public" (default) or "private"
network_name: ""         # Required when network_mode=private: name of Hetzner private network
```

### Supported Server Types

Examples of available Hetzner server types:

| Type | vCPUs | RAM | Disk |
|------|-------|-----|------|
| `cx22` | 2 | 4 GB | 40 GB |
| `cx32` | 4 | 8 GB | 80 GB |
| `cx42` | 8 | 16 GB | 160 GB |
| `cx52` | 16 | 32 GB | 320 GB |
| `cpx21` | 3 | 4 GB | 80 GB |
| `cpx31` | 4 | 8 GB | 160 GB |

See the [Hetzner Cloud pricing page](https://www.hetzner.com/cloud/#pricing) for the full list.

### Supported Locations

| Location | Description |
|----------|-------------|
| `fsn1` | Falkenstein, Germany |
| `nbg1` | Nuremberg, Germany |
| `hel1` | Helsinki, Finland |
| `ash` | Ashburn, VA, USA |
| `hil` | Hillsboro, OR, USA |
| `sin` | Singapore |

## Networking Modes

### Public IP Mode (default)

The server receives public IPv4 and IPv6 addresses automatically:

```yaml
network_mode: public
```

### Private VNet Mode

The server is attached to an existing Hetzner private network and receives no public IP:

```yaml
network_mode: private
network_name: my-internal-network
```

## Architecture

The provider follows the [Sidero Omni infra provider architecture](https://github.com/siderolabs/omni/tree/main/client/pkg/infra):

```
cmd/omni-infra-provider-hetzner/   # Main binary entry point
internal/pkg/
  config/                          # Multi-project configuration
  hcloud/                          # Resilient Hetzner API client wrapper
  provider/
    meta/                          # Provider ID
    resources/                     # COSI machine state resource
    data.go                        # Per-machine provider schema
    provision.go                   # VM lifecycle provisioner
api/specs/                         # Protobuf state definitions
```
