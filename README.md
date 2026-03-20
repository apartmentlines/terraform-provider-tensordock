# Terraform Provider for TensorDock

This repository contains a Terraform provider plugin for managing TensorDock resources through the public TensorDock v2 REST API.

## Scope

The implementation follows the public TensorDock v2 API surface verified from the reference CLI:

- provider authentication with a bearer token
- location and hostnode discovery
- secret management
- location-based and hostnode-based instance creation
- instance read / refresh
- in-place instance resize through the documented `PUT /instances/{id}/modify` endpoint
- instance start / stop operations
- instance deletion
- resource import by instance ID or secret ID

## Supported resources

- `tensordock_instance`
- `tensordock_secret`

## Supported ephemeral resources

- `tensordock_secret_value`

## Supported data sources

- `tensordock_locations`
- `tensordock_hostnodes`

## Current limitations

- **Some instance create-time fields are replace-only.** `image`, `location_id`, `hostnode_id`, `use_dedicated_ip`, `port_forwards`, `ssh_public_key`, and `cloud_init_json` are treated as replacement triggers because the public instance management API does not document in-place mutation for them.
- **Drift visibility is partial for create-only fields.** The documented `GET /instances/{id}` response returns current runtime attributes such as status, IP, resources, port forwards, and hourly rate, but it does not document returning `image`, `location_id`, SSH key injection state, or cloud-init payload. Those fields remain stored in Terraform state but cannot be fully re-read from the public API.
- **Secret values are management-only.** `tensordock_secret.value` is write-only and is never stored in Terraform state, including after creation or import.
- **Ephemeral secret values require Terraform 1.11+.** `tensordock_secret_value` and the write-only `ssh_public_key` flow rely on Terraform `>= 1.11.0`.

## Provider configuration

```hcl
provider "tensordock" {
  api_token = var.tensordock_api_token
}
```

Supported provider arguments:

- `api_token` (optional, sensitive) — can also be supplied with `TENSORDOCK_API_TOKEN`
- `base_url` (optional) — defaults to `https://dashboard.tensordock.com/api/v2`, can also be supplied with `TENSORDOCK_BASE_URL`

## Example usage

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
  name             = "gpu-worker-1"
  image            = "ubuntu2404"
  location_id      = "loc-uuid-12345"
  vcpu_count       = 8
  ram_gb           = 32
  storage_gb       = 200
  gpu_type         = "geforcertx4090-pcie-24gb"
  gpu_count        = 1
  use_dedicated_ip = true
  port_forwards = [{
    internal_port = 22
    external_port = 22022
  }]
  ssh_public_key   = file("~/.ssh/id_ed25519.pub")
  power_state      = "running"

  cloud_init_json = jsonencode({
    package_update  = true
    package_upgrade = false
    packages        = ["curl", "git"]
    runcmd          = [
      "echo hello from TensorDock",
      "apt-get install -y nginx",
    ]
  })
}

resource "tensordock_secret" "deploy_key" {
  name  = "deploy-key"
  type  = "ssh"
  value = file("~/.ssh/id_ed25519")
}

ephemeral "tensordock_secret_value" "deploy_key" {
  secret_id = tensordock_secret.deploy_key.id
}

resource "tensordock_instance" "gpu_worker_from_secret_value" {
  name           = "gpu-worker-2"
  image          = "ubuntu2404"
  location_id    = "loc-uuid-12345"
  vcpu_count     = 8
  ram_gb         = 32
  storage_gb     = 200
  gpu_type       = "geforcertx4090-pcie-24gb"
  gpu_count      = 1
  ssh_public_key = ephemeral.tensordock_secret_value.deploy_key.value
  power_state    = "running"
}

data "tensordock_locations" "all" {}
data "tensordock_hostnodes" "all" {}
```

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

The provider version is embedded from the repo-root `VERSION` file. Update that file, then run `go build` or `go install` without needing manual `-ldflags`.

The module path and provider registry address are currently set to `apartmentlines/tensordock`. Update them if you plan to publish under a different namespace.
