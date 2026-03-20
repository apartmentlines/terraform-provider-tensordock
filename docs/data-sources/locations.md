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

output "location_ids" {
  value = [
    for location in data.tensordock_locations.all.locations : {
      id      = location.id
      city    = location.city
      country = location.country
    }
  ]
}
```

## Attributes Reference

- `locations` (List of Objects) Locations returned by `GET /locations`.
- `locations[*].id` (String) Location UUID.
- `locations[*].city` (String) City name.
- `locations[*].stateprovince` (String) State or province name.
- `locations[*].country` (String) Country name.
- `locations[*].tier` (Number) TensorDock location tier.
- `locations[*].gpus` (List of Objects) GPU offerings available in the location.
- `locations[*].gpus[*].v0_name` (String) GPU model identifier to use with `tensordock_instance.gpu_type`.
- `locations[*].gpus[*].display_name` (String) Human-readable GPU model name.
- `locations[*].gpus[*].max_count` (Number) Maximum GPU count exposed for that model in the location response.
- `locations[*].gpus[*].price_per_hr` (Number) GPU hourly price component.
- `locations[*].gpus[*].resources` (Object) Maximum vCPU, RAM, and storage limits associated with the GPU offering.
- `locations[*].gpus[*].pricing` (Object) Per-vCPU, per-RAM, and per-storage pricing components.
- `locations[*].gpus[*].network_features` (Object) Dedicated IP, port forwarding, and network storage availability flags.
