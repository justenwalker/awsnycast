resource "aws_key_pair" "awsnycast" {
  key_name   = "awsnycast-ssh-key"
  public_key = local.ssh_public_key
}

locals {
  ssh_private_key = abspath("${path.module}/id_rsa")
  ssh_public_key  = trimspace(file(abspath("${path.module}/id_rsa.pub")))
}

output "ssh_private_key" {
  value = local.ssh_private_key
}

output "ssh_public_key" {
  value = local.ssh_public_key
}