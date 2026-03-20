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
  type             = "ssh"
  value_wo         = file("~/.ssh/id_ed25519")
  value_wo_version = 1
}
```

## Argument Reference

- `name` (String) Secret name.
- `type` (String) Secret type.
- `value_wo` (String, Sensitive) Secret value used during creation or replacement. This attribute is write-only and is not stored in Terraform state.
- `value_wo_version` (Number) Rotation trigger that Terraform persists. Increment this value whenever `value_wo` changes.

## Attributes Reference

- `id` (String) TensorDock secret ID.

## Import

Import is supported with the TensorDock secret ID:

```bash
terraform import tensordock_secret.example <secret-id>
```

`value_wo` is write-only and is not recovered during import or refresh. Terraform relies on `value_wo_version` to detect rotations.
