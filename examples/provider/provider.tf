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
