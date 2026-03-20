terraform {
  required_version = ">= 1.11.0"

  required_providers {
    tensordock = {
      source = "apartmentlines/tensordock"
    }
  }
}

resource "tensordock_instance" "example" {
  name        = "gpu-worker-1"
  image       = "ubuntu2404"
  location_id = "loc-uuid-12345"

  vcpu_count = 8
  ram_gb     = 32
  storage_gb = 200
  gpu_type   = "geforcertx4090-pcie-24gb"
  gpu_count  = 1

  use_dedicated_ip = true
  ssh_public_key   = file("~/.ssh/id_ed25519.pub")
  power_state      = "running"
}
