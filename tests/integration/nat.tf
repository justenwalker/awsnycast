locals {
  awsnycast_bin       = abspath("${path.module}/awsnycast")
  nat_config_template = abspath("${path.module}/templates/nat.conf")
  playbook_dir        = abspath("${path.module}/files/playbook")
  playbook_zip        = abspath("${path.module}/files/playbook.zip")
}

data "archive_file" "playbook" {
  type             = "zip"
  source_dir       = local.playbook_dir
  output_path      = local.playbook_zip
  output_file_mode = "0666"
}

resource "aws_instance" "nat-a" {
  ami                    = local.ami
  instance_type          = local.instance_type
  source_dest_check      = false
  key_name               = aws_key_pair.awsnycast.key_name
  subnet_id              = aws_subnet.publica.id
  vpc_security_group_ids = [aws_security_group.allow_all.id]
  tags = {
    Name = "awsnycast nat us-west-2a"
  }
  user_data = base64gzip(templatefile(local.nat_config_template, {
    playbook_content  = filebase64(data.archive_file.playbook.output_path)
    vpc_cidr          = local.vpc_cidr_block
    vpc_id            = aws_vpc.main.id
    availability_zone = aws_subnet.publica.availability_zone
  }))
  iam_instance_profile = aws_iam_instance_profile.test_profile.id
  provisioner "file" {
    source      = local.awsnycast_bin
    destination = "/home/ubuntu/awsnycast"
    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(local.ssh_private_key)
      host        = self.public_ip
    }
  }
  provisioner "remote-exec" {
    inline = [
      "sudo cp /home/ubuntu/awsnycast /usr/local/bin/awsnycast",
      "sudo chmod 755 /usr/local/bin/awsnycast",
      "while sudo pkill -0 cloud-init; do sleep 2; done",
    ]
    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(local.ssh_private_key)
      host        = self.public_ip
    }
  }
}

resource "aws_instance" "nat-b" {
  ami                    = local.ami
  instance_type          = local.instance_type
  source_dest_check      = false
  key_name               = aws_key_pair.awsnycast.key_name
  subnet_id              = aws_subnet.publicb.id
  vpc_security_group_ids = [aws_security_group.allow_all.id]
  tags = {
    Name = "awsnycast nat us-west-2b"
  }
  user_data = base64gzip(templatefile(local.nat_config_template, {
    playbook_content  = filebase64(data.archive_file.playbook.output_path)
    vpc_cidr          = local.vpc_cidr_block
    vpc_id            = aws_vpc.main.id
    availability_zone = aws_subnet.publicb.availability_zone
  }))
  iam_instance_profile = aws_iam_instance_profile.test_profile.id
  provisioner "file" {
    source      = local.awsnycast_bin
    destination = "/home/ubuntu/awsnycast"
    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(local.ssh_private_key)
      host        = self.public_ip
    }
  }
  provisioner "remote-exec" {
    inline = [
      "sudo cp /home/ubuntu/awsnycast /usr/local/bin/awsnycast",
      "sudo chmod 755 /usr/local/bin/awsnycast",
      "while sudo pkill -0 cloud-init; do sleep 2; done",
    ]
    connection {
      type        = "ssh"
      user        = "ubuntu"
      private_key = file(local.ssh_private_key)
      host        = self.public_ip
    }
  }
}
