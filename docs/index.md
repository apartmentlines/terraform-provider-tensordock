---
page_title: "TensorDock Provider"
subcategory: "Compute"
description: |-
  Terraform provider for managing TensorDock resources through the public v2 REST API.
---

# TensorDock Provider

The TensorDock provider authenticates to the public TensorDock v2 API with a bearer token.

Supported resources:

- [`tensordock_instance`](resources/instance.md)
- [`tensordock_secret`](resources/secret.md)

Supported ephemeral resources:

- [`tensordock_secret_value`](ephemeral-resources/secret_value.md)

Supported data sources:

- [`tensordock_locations`](data-sources/locations.md)
- [`tensordock_hostnodes`](data-sources/hostnodes.md)

## Example

```terraform
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

## Compatibility

- `tensordock_secret_value` requires Terraform `>= 1.11.0`.
- Direct instance creation can pass `ssh_public_key` from a file or from `ephemeral.tensordock_secret_value.<name>.value`.

## Schema

### Optional

- `api_token` (String, Sensitive) TensorDock API token. Can also be set with `TENSORDOCK_API_TOKEN`.
- `base_url` (String) TensorDock API base URL. Defaults to `https://dashboard.tensordock.com/api/v2`. Can also be set with `TENSORDOCK_BASE_URL`.
