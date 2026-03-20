# Terraform Provider for TensorDock

This repository contains a Terraform provider for managing [TensorDock](https://www.tensordock.com) resources through the public [TensorDock v2 REST API](https://dashboard.tensordock.com/api/docs).

## Supported objects

- Resources: `tensordock_instance`, `tensordock_secret`
- Ephemeral resources: `tensordock_secret_value`
- Data sources: `tensordock_locations`, `tensordock_hostnodes`

Main docs: [Provider docs](docs/index.md)

Examples: [examples/](examples/)

## Requirements

- Terraform `>= 1.11.0` if you use `tensordock_secret_value` or want to pass an ephemeral value into the write-only `ssh_public_key` argument.

## Provider configuration

```hcl
terraform {
  required_providers {
    tensordock = {
      source = "apartmentlines/tensordock"
    }
  }
}

provider "tensordock" {
  api_token = var.tensordock_api_token
}
```

Supported provider arguments:

- `api_token` (optional, sensitive): TensorDock API token. Can also be supplied with `TENSORDOCK_API_TOKEN` environment variable.
- `base_url` (optional): TensorDock API base URL. Defaults to `https://dashboard.tensordock.com/api/v2`. Can also be supplied with `TENSORDOCK_BASE_URL`.

## Quickstart

```hcl
terraform {
  required_version = ">= 1.11.0"

  required_providers {
    tensordock = {
      source = "apartmentlines/tensordock"
    }
  }
}

provider "tensordock" {
  api_token = var.tensordock_api_token
}

resource "tensordock_instance" "gpu_worker" {
  name        = "gpu-worker-1"
  image       = "ubuntu2404"
  location_id = "loc-uuid-12345"

  vcpu_count = 8
  ram_gb     = 32
  storage_gb = 200
  gpu_type   = "geforcertx4090-pcie-24gb"
  gpu_count  = 1

  ssh_public_key = file("~/.ssh/id_ed25519.pub")
  power_state    = "running"
}
```

For the full argument and behavior reference, see the [main docs page](docs/index.md). Example configurations live in [examples/](examples/).

## Placement model

- Use `location_id` for location-based deployment. This requires `gpu_type` and `gpu_count >= 1`.
- Use `hostnode_id` for direct hostnode deployment. Hostnode deployments may omit GPU settings.
- Set exactly one of `location_id` or `hostnode_id`.

The `tensordock_locations` and `tensordock_hostnodes` data sources expose the IDs and capacity details needed to choose placement.

## State and write-only behavior

- `tensordock_secret.value_wo` is write-only and is never stored in Terraform state, including after import.
- `tensordock_instance.ssh_public_key` is write-only and is never stored in Terraform state.
- `tensordock_secret_value` fetches a secret value for the current Terraform run only; its `value` is not persisted to state.
- When rotating a secret managed by `tensordock_secret`, increment `value_wo_version`.

## Operational limits

- `image`, `location_id`, `hostnode_id`, `use_dedicated_ip`, `port_forwards`, `ssh_public_key`, and `cloud_init_json` are create-time fields and force replacement when changed.
- `storage_gb` must be at least `100`.
- Non-Windows images require `ssh_public_key` during creation.
- `power_state` must be `running` or `stopped`.
- `cloud_init_json`, if set, must decode to a JSON object.
- Port forward values must use ports between `1` and `65535`.
- In-place instance resize supports CPU, RAM, storage growth, and GPU changes through the documented modify endpoint, but:
  - storage cannot be shrunk
  - GPUs cannot be removed in place
  - CPU changes must use a multiple of 2 cores
  - RAM changes must use a TensorDock-supported modify size
- Drift visibility is partial for some create-time fields because the public instance read API does not re-return every creation input.

## Import

```bash
terraform import tensordock_instance.gpu_worker <instance-id>
terraform import tensordock_secret.deploy_key <secret-id>
```

## Build

This provider is implemented with HashiCorp's Terraform Plugin Framework.

```bash
go mod tidy
go test ./...
go install
```

The provider version is embedded from the repo-root `VERSION` file. Update that file, then run `go build` or `go install` without manual `-ldflags`.

The module path and provider registry address are currently set to `apartmentlines/tensordock`. Update them if you plan to publish under a different namespace.
