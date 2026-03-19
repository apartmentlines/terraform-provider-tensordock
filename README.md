# Terraform Provider for TensorDock

This repository contains a Terraform provider plugin for provisioning and managing TensorDock virtual machine instances through the public TensorDock v2 REST API.

## Scope

The implementation is intentionally limited to the instance lifecycle surface that is explicitly documented by TensorDock's public API:

- provider authentication with a bearer token
- location-based instance creation
- instance read / refresh
- in-place instance resize through the documented `PUT /instances/{id}/modify` endpoint
- instance start / stop operations
- instance deletion
- resource import by instance ID

## Supported resource

- `tensordock_instance`

## Current limitations

The provider is conservative by design and only models fields that are explicitly documented.

- **Location-based deployment only.** The public documentation shows a concrete create payload for `location_id`, but does not publish a hostnode-based create request payload. Hostnode placement is therefore intentionally omitted.
- **No create-time port forward management yet.** TensorDock documents port forwarding in hostnode discovery and in product docs, but the public create endpoint example does not publish the corresponding request shape.
- **No data sources yet.** Discovery of locations and hostnodes remains a pre-Terraform step in this MVP.
- **Some create-time fields are replace-only.** `image`, `location_id`, `use_dedicated_ip`, `ssh_public_key`, and `cloud_init_json` are treated as replacement triggers because the public instance management API does not document in-place mutation for them.
- **Drift visibility is partial for create-only fields.** The documented `GET /instances/{id}` response returns current runtime attributes such as status, IP, resources, port forwards, and hourly rate, but it does not document returning `image`, `location_id`, SSH key injection state, or cloud-init payload. Those fields remain stored in Terraform state but cannot be fully re-read from the public API.

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
  name           = "gpu-worker-1"
  image          = "ubuntu2404"
  location_id    = "loc-uuid-12345"
  vcpu_count     = 8
  ram_gb         = 32
  storage_gb     = 200
  gpu_type       = "geforcertx4090-pcie-24gb"
  gpu_count      = 1
  use_dedicated_ip = true
  ssh_public_key = file("~/.ssh/id_ed25519.pub")
  power_state    = "running"

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
```

## Import

```bash
terraform import tensordock_instance.gpu_worker <instance-id>
```

## Build

This provider is implemented with HashiCorp's Terraform Plugin Framework.

```bash
go mod tidy
go test ./...
go install
```

The module path and provider registry address are currently set to `apartmentlines/tensordock`. Update them if you plan to publish under a different namespace.
