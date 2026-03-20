---
page_title: "tensordock_instance Resource"
subcategory: "Compute"
description: |-
  Manages a TensorDock virtual machine instance.
---

# tensordock_instance

Manages a TensorDock virtual machine instance using the public instance creation and instance management endpoints.

## Example Usage

```terraform
terraform {
  required_version = ">= 1.11.0"
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

  use_dedicated_ip = true
  port_forwards = [{
    internal_port = 22
    external_port = 22022
  }]
  ssh_public_key = file("~/.ssh/id_ed25519.pub")
  power_state    = "running"
}

resource "tensordock_secret" "deploy_key" {
  name             = "deploy-key"
  type             = "SSHKEY"
  value_wo         = file("~/.ssh/id_ed25519.pub")
  value_wo_version = 1
}

ephemeral "tensordock_secret_value" "gpu_worker_ssh" {
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
  ssh_public_key = ephemeral.tensordock_secret_value.gpu_worker_ssh.value
  power_state    = "running"
}
```

## Argument Reference

### Required

- `name` (String) Instance name.
- `image` (String) TensorDock image identifier.
- `vcpu_count` (Number) Requested vCPU count.
- `ram_gb` (Number) Requested memory in GiB.
- `storage_gb` (Number) Requested storage in GiB. TensorDock documents a minimum of 100GB.

### Placement

- `location_id` (String) TensorDock location UUID for location-based deployment. Exactly one of `location_id` or `hostnode_id` must be set.
- `hostnode_id` (String) TensorDock hostnode UUID for direct hostnode deployment. Exactly one of `location_id` or `hostnode_id` must be set.

### GPU

- `gpu_type` (String) TensorDock GPU model `v0Name`. Required for location-based deployments.
- `gpu_count` (Number) Number of GPUs of `gpu_type`. Required for location-based deployments.

### Optional

- `use_dedicated_ip` (Boolean) Request a dedicated IP during creation. Replace-only.
- `port_forwards` (List of Objects) Optional create-time port forward mappings. Replace-only.
- `ssh_public_key` (String, Sensitive) SSH public key injected during instance creation. This attribute is write-only and is not stored in Terraform state. Required for non-Windows images. Replace-only.
- `cloud_init_json` (String) JSON representation of TensorDock's documented `cloud_init` object. Replace-only.
- `power_state` (String) Desired power state. Valid values are `running` and `stopped`.

## Attributes Reference

In addition to the arguments above, the following attributes are exported:

- `id` (String) TensorDock instance ID.
- `status` (String) Raw status returned by TensorDock.
- `ip_address` (String) Instance IP address.
- `rate_hourly` (Number) Hourly rate returned by TensorDock.

## Import

Import is supported with the TensorDock instance ID:

```bash
terraform import tensordock_instance.example <instance-id>
```
