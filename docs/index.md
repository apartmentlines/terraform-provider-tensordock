---
page_title: "TensorDock Provider"
subcategory: "Compute"
description: |-
  Terraform provider for provisioning TensorDock virtual machine instances through the public v2 REST API.
---

# TensorDock Provider

The TensorDock provider authenticates to the public TensorDock v2 API with a bearer token.

## Example

```terraform
provider "tensordock" {
  api_token = var.tensordock_api_token
}
```

## Schema

### Optional

- `api_token` (String, Sensitive) TensorDock API token. Can also be set with `TENSORDOCK_API_TOKEN`.
- `base_url` (String) TensorDock API base URL. Defaults to `https://dashboard.tensordock.com/api/v2`. Can also be set with `TENSORDOCK_BASE_URL`.
