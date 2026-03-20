---
page_title: "tensordock_locations Data Source"
subcategory: "Compute"
description: |-
  Lists TensorDock locations.
---

# tensordock_locations

Fetches all locations returned by the public TensorDock API.

## Example Usage

```terraform
data "tensordock_locations" "all" {}
```

## Attributes Reference

- `locations` (List of Objects) Locations returned by `GET /locations`.
