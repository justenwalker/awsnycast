locals {
  internal_config_template = abspath("${path.module}/templates/internal.conf")
}
resource "aws_instance" "internal-a" {
  ami                    = local.ami
  instance_type          = local.instance_type
  key_name               = aws_key_pair.awsnycast.key_name
  subnet_id              = aws_subnet.privatea.id
  vpc_security_group_ids = [aws_security_group.allow_all.id]
  tags = {
    Name = "awsnycast internal us-west-2a"
  }
  user_data = base64gzip(templatefile(local.internal_config_template, { network_prefix = "10.0" }))
  provisioner "remote-exec" {
    inline = [
      "while sudo pkill -0 cloud-init; do sleep 2; done"
    ]
    connection {
      type                = "ssh"
      user                = "ubuntu"
      host                = self.private_dns
      private_key         = file(local.ssh_private_key)
      bastion_user        = "ubuntu"
      bastion_host        = aws_instance.nat-a.public_ip
      bastion_private_key = file(local.ssh_private_key)
    }
  }
}

resource "aws_instance" "internal-b" {
  ami                    = local.ami
  instance_type          = local.instance_type
  key_name               = aws_key_pair.awsnycast.key_name
  subnet_id              = aws_subnet.privateb.id
  vpc_security_group_ids = [aws_security_group.allow_all.id]
  tags = {
    Name = "awsnycast internal us-west-2b"
  }
  user_data = base64gzip(templatefile(local.internal_config_template, { network_prefix = "10.0" }))
  provisioner "remote-exec" {
    inline = [
      "while sudo pkill -0 cloud-init; do sleep 2; done"
    ]
    connection {
      type                = "ssh"
      user                = "ubuntu"
      host                = self.private_dns
      private_key         = file(local.ssh_private_key)
      bastion_user        = "ubuntu"
      bastion_host        = aws_instance.nat-b.public_ip
      bastion_private_key = file(local.ssh_private_key)
    }
  }
}
