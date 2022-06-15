output "internal_private_ips" {
  value = [aws_instance.internal-a.private_ip, aws_instance.internal-b.private_ip]
}

output "nat_public_ips" {
  value = [aws_instance.nat-a.public_ip, aws_instance.nat-b.public_ip]
}
