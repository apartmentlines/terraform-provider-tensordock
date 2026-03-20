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

output "hostnode_capacity" {
  value = [
    for hostnode in data.tensordock_hostnodes.all.hostnodes : {
      id                = hostnode.id
      location_id       = hostnode.location_id
      available_vcpus   = hostnode.available_resources.vcpu_count
      available_ram_gb  = hostnode.available_resources.ram_gb
      public_ip_ready   = hostnode.available_resources.has_public_ip_available
      available_gpu_ids = [for gpu in hostnode.available_resources.gpus : gpu.v0_name]
    }
  ]
}
```

## Attributes Reference

- `hostnodes` (List of Objects) Hostnodes returned by `GET /hostnodes`.
- `hostnodes[*].id` (String) Hostnode UUID.
- `hostnodes[*].location_id` (String) Parent location UUID.
- `hostnodes[*].engine` (String) Hostnode engine label from the API.
- `hostnodes[*].uptime_percentage` (Number) Reported uptime percentage.
- `hostnodes[*].available_resources` (Object) Available capacity and placement-related fields.
- `hostnodes[*].available_resources.gpus` (List of Objects) GPU availability entries for the hostnode.
- `hostnodes[*].available_resources.gpus[*].v0_name` (String) GPU model identifier usable with `tensordock_instance.gpu_type`.
- `hostnodes[*].available_resources.gpus[*].available_count` (Number) Currently available count for that GPU model.
- `hostnodes[*].available_resources.vcpu_count` (Number) Available vCPU count.
- `hostnodes[*].available_resources.ram_gb` (Number) Available RAM in GiB.
- `hostnodes[*].available_resources.storage_gb` (Number) Available storage in GiB.
- `hostnodes[*].available_resources.available_ports` (List of Numbers) Available public ports reported by the API.
- `hostnodes[*].available_resources.has_public_ip_available` (Boolean) Whether the hostnode currently reports public IP availability.
- `hostnodes[*].pricing` (Object) Per-vCPU, per-RAM, and per-storage pricing components.
- `hostnodes[*].location` (Object) Expanded location metadata for the hostnode.
