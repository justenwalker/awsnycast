resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true
  tags = {
    Name = "awsnycast vpc"
  }
}

locals {
  vpc_cidr_block = aws_vpc.main.cidr_block
}

resource "aws_internet_gateway" "gw" {
  vpc_id = aws_vpc.main.id

  tags = {
    Name = "awsnycast igw"
  }
}

resource "aws_route_table" "public" {
  vpc_id = aws_vpc.main.id
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gw.id
  }

  tags = {
    Name = "awsnycast public"
  }
}

resource "aws_route_table" "privatea" {
  vpc_id = aws_vpc.main.id
  # N.B. These type of routes are the equivalent of what AWSnycast creates in this configuration
  #      This allows you to comment / un-comment them to play with what happens
  #    route { 
  #        cidr_block = "0.0.0.0/0"
  #        instance_id = "${aws_instance.nat-a.id}"
  #    }
  #    route {
  #        cidr_block = "192.168.1.1/32"
  #        instance_id = "${aws_instance.nat-a.id}"
  #    }
  tags = {
    Name = "awsnycast private a"
    az   = "us-west-2a"
    type = "private"
  }
}

resource "aws_route_table" "privateb" {
  vpc_id = aws_vpc.main.id
  #    route {
  #        cidr_block = "0.0.0.0/0"
  #        instance_id = "${aws_instance.nat-b.id}"
  #    }
  #    route {
  #        cidr_block = "192.168.1.1/32"
  #        instance_id = "${aws_instance.nat-b.id}"
  #    }
  tags = {
    Name = "awsnycast private b"
    az   = "us-west-2b"
    type = "private"
  }
}

resource "aws_subnet" "publica" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.0.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2a"

  tags = {
    Name = "awsnycast public a"
    az   = "us-west-2a"
    type = "public"
  }
}

resource "aws_subnet" "publicb" {
  vpc_id                  = aws_vpc.main.id
  cidr_block              = "10.0.1.0/24"
  map_public_ip_on_launch = true
  availability_zone       = "us-west-2b"

  tags = {
    Name = "awsnycast public b"
    az   = "us-west-2a"
    type = "public"
  }
}

resource "aws_route_table_association" "publica" {
  subnet_id      = aws_subnet.publica.id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "publicb" {
  subnet_id      = aws_subnet.publicb.id
  route_table_id = aws_route_table.public.id
}

resource "aws_subnet" "privatea" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.10.0/24"
  availability_zone = "us-west-2a"

  tags = {
    Name    = "awsnycast us-west-2a private"
    private = "true"
  }
}

resource "aws_subnet" "privateb" {
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.11.0/24"
  availability_zone = "us-west-2b"

  tags = {
    Name    = "awsnycast us-west-2b private"
    private = "true"
  }
}

resource "aws_route_table_association" "privatea" {
  subnet_id      = aws_subnet.privatea.id
  route_table_id = aws_route_table.privatea.id
}

resource "aws_route_table_association" "privateb" {
  subnet_id      = aws_subnet.privateb.id
  route_table_id = aws_route_table.privateb.id
}

