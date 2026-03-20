terraform {
  required_version = ">= 1.11.0"

  required_providers {
    tensordock = {
      source = "apartmentlines/tensordock"
    }
  }
}

resource "tensordock_secret" "example" {
  name             = "deploy-key"
  type             = "SSHKEY"
  value_wo         = file("~/.ssh/id_ed25519.pub")
  value_wo_version = 1
}

ephemeral "tensordock_secret_value" "example" {
  secret_id = tensordock_secret.example.id
}

resource "tensordock_instance" "gpu_worker" {
  name        = "gpu-worker-1"
  image       = "ubuntu2404"
  location_id = "loc-uuid-12345"

  vcpu_count = 8
  ram_gb     = 32
  storage_gb = 200
  gpu_type   = "geforcertx4090-pcie-24gb"
  gpu_count  = 1

  ssh_public_key = ephemeral.tensordock_secret_value.example.value
  power_state    = "running"
}
