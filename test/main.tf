terraform {
  required_providers {
    podman = {
      source = "registry.terraform.io/cmp0st/podman"
    }
  }
}

provider "podman" {}

resource "podman_secret" "example" {
  name   = "foo"
  secret = "bar"
  driver = "file"
  labels = {
    "foo" = "bar"
  }
}
