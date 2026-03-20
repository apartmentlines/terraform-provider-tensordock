---
page_title: "tensordock_instance Resource"
subcategory: "Compute"
description: |-
  Manages a TensorDock virtual machine instance.
---

# tensordock_instance

Manages a TensorDock virtual machine instance using the public instance creation and instance management endpoints.

## Example Usage

Create an instance with a direct SSH public key:

```terraform
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
```

Read a managed secret value ephemerally and pass it into `ssh_public_key`:

```terraform
terraform {
  required_version = ">= 1.11.0"
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
  name        = "gpu-worker-2"
  image       = "ubuntu2404"
  location_id = "loc-uuid-12345"

  vcpu_count = 8
  ram_gb     = 32
  storage_gb = 200
  gpu_type   = "geforcertx4090-pcie-24gb"
  gpu_count  = 1

  ssh_public_key = ephemeral.tensordock_secret_value.gpu_worker_ssh.value
  power_state    = "running"
}
```

## Behavior and constraints

- Set exactly one of `location_id` or `hostnode_id`.
- Location-based deployment requires both `gpu_type` and `gpu_count >= 1`.
- Hostnode-based deployment may omit GPU settings.
- `storage_gb` must be at least `100`.
- `power_state` must be either `running` or `stopped`.
- `cloud_init_json`, if set, must decode to a JSON object.
- `port_forwards` values must use ports between `1` and `65535`.
- `ssh_public_key` is required for non-Windows images during creation.
- `ssh_public_key` is write-only and is not stored in Terraform state.

## Argument Reference

### Required

- `name` (String) Instance name.
- `image` (String) TensorDock image identifier, for example `ubuntu2404`.
- `vcpu_count` (Number) Requested vCPU count.
- `ram_gb` (Number) Requested memory in GiB.
- `storage_gb` (Number) Requested storage in GiB. TensorDock documents a minimum of 100GB.

### Placement

- `location_id` (String) TensorDock location UUID for location-based deployment. Exactly one of `location_id` or `hostnode_id` must be set.
- `hostnode_id` (String) TensorDock hostnode UUID for direct hostnode deployment. Exactly one of `location_id` or `hostnode_id` must be set.

### GPU

- `gpu_type` (String) TensorDock GPU model `v0_name`, for example `geforcertx4090-pcie-24gb`. Required for location-based deployments.
- `gpu_count` (Number) Number of GPUs of `gpu_type`. Required for location-based deployments.

### Optional

- `use_dedicated_ip` (Boolean) Request a dedicated IP during creation. This is a create-time field and changing it forces replacement.
- `port_forwards` (List of Objects) Optional create-time port forward mappings. TensorDock also returns current port forwards on instance reads. Changing this field forces replacement.
- `ssh_public_key` (String, Sensitive) SSH public key injected during instance creation. This attribute is write-only and is not stored in Terraform state. Required for non-Windows images. Changing this field forces replacement.
- `cloud_init_json` (String) JSON representation of TensorDock's documented `cloud_init` object. This is a create-time field and changing it forces replacement.
- `power_state` (String) Desired power state. Valid values are `running` and `stopped`.

## Update behavior

- `image`, `location_id`, `hostnode_id`, `use_dedicated_ip`, `port_forwards`, `ssh_public_key`, and `cloud_init_json` are create-time fields and force replacement when changed.
- In-place resize uses the documented modify endpoint. The provider stops the instance before modification when required, then reconciles the instance back to the requested `power_state`.
- `storage_gb` can increase in place, but it cannot be reduced in place.
- GPU removal is not supported in place. Recreate the instance to remove GPUs.
- CPU changes through the modify endpoint must use a multiple of `2` cores.
- RAM changes through the modify endpoint must use a TensorDock-supported modify size.

## State and drift notes

- The public instance read API returns current runtime attributes such as status, IP address, resources, port forwards, and hourly rate.
- Some create-time fields are not fully re-read from the public API, including `image`, placement choice, SSH key injection state, and `cloud_init_json`. Terraform retains those values in state, but full drift detection for them is not available.

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
