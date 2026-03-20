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
  name  = "deploy-key"
  type  = "ssh"
  value = file("~/.ssh/id_ed25519")
}
```

## Argument Reference

- `name` (String) Secret name.
- `type` (String) Secret type.
- `value` (String, Sensitive) Secret value used during creation or replacement.

## Attributes Reference

- `id` (String) TensorDock secret ID.

## Import

Import is supported with the TensorDock secret ID:

```bash
terraform import tensordock_secret.example <secret-id>
```

`value` is write-only and is not recovered during import or refresh.
