---
page_title: "tensordock_secret Resource"
subcategory: "Security"
description: |-
  Manages a TensorDock secret.
---

# tensordock_secret

Manages a TensorDock secret using the public secrets API.

## Example Usage

```terraform
resource "tensordock_secret" "deploy_key" {
  name             = "deploy-key"
  type             = "SSHKEY"
  value_wo         = file("~/.ssh/id_ed25519.pub")
  value_wo_version = 1
}
```

## Behavior notes

- `name`, `type`, and `value_wo` must be supplied when creating a secret.
- `value_wo` is write-only and is not stored in Terraform state.
- If `value_wo` is set during creation and `value_wo_version` is omitted, the provider defaults `value_wo_version` to `1`.
- Increment `value_wo_version` whenever `value_wo` changes so Terraform can detect a rotation and replace the secret.
- Import and refresh do not recover `value_wo`.

## Argument Reference

- `name` (String) Secret name.
- `type` (String) Secret type.
- `value_wo` (String, Sensitive) Secret value used during creation or replacement. This attribute is write-only and is not stored in Terraform state.
- `value_wo_version` (Number) Rotation token that Terraform persists. Increment this value whenever `value_wo` changes.

## Attributes Reference

- `id` (String) TensorDock secret ID.

## Import

Import is supported with the TensorDock secret ID:

```bash
terraform import tensordock_secret.example <secret-id>
```
