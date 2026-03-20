---
page_title: "tensordock_secret_value Ephemeral Resource"
subcategory: "Security"
description: |-
  Fetches a TensorDock secret value for use during a Terraform run without persisting it to state.
---

# tensordock_secret_value

Fetches a TensorDock secret value from the public secrets API for use during the current Terraform run. The returned `value` is ephemeral and is not stored in Terraform state.

## Example Usage

```terraform
terraform {
  required_version = ">= 1.11.0"
}

resource "tensordock_secret" "deploy_key" {
  name = "deploy-key"
  type = "SSHKEY"
}

ephemeral "tensordock_secret_value" "deploy_key" {
  secret_id = tensordock_secret.deploy_key.id
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
  ssh_public_key = ephemeral.tensordock_secret_value.deploy_key.value
  power_state    = "running"
}
```

## Argument Reference

- `secret_id` (String) TensorDock secret ID to fetch.

## Attributes Reference

- `id` (String) TensorDock secret ID.
- `name` (String) Secret name.
- `type` (String) Secret type.
- `value` (String, Sensitive) Secret value fetched live from the TensorDock API for the current run only.
