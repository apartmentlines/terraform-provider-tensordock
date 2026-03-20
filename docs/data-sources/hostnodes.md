---
page_title: "tensordock_hostnodes Data Source"
subcategory: "Compute"
description: |-
  Lists TensorDock hostnodes.
---

# tensordock_hostnodes

Fetches all hostnodes returned by the public TensorDock API.

## Example Usage

```terraform
data "tensordock_hostnodes" "all" {}
```

## Attributes Reference

- `hostnodes` (List of Objects) Hostnodes returned by `GET /hostnodes`.
