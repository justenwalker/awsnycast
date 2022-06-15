data "aws_iam_policy_document" "assume_role" {
  statement {
    effect  = "Allow"
    actions = ["sts:AssumeRole"]
    principals {
      identifiers = ["ec2.amazonaws.com"]
      type        = "Service"
    }
  }
}
data "aws_iam_policy_document" "modify_routes_policy" {
  statement {
    effect = "Allow"
    actions = [
      "ec2:ReplaceRoute",
      "ec2:CreateRoute",
      "ec2:DeleteRoute",
      "ec2:DescribeRouteTables",
      "ec2:DescribeNetworkInterfaces",
      "ec2:DescribeInstanceAttribute",
      "ec2:DescribeInstanceStatus",
    ]
    resources = ["*"]
  }
}
resource "aws_iam_role" "role" {
  name               = "awsnycast_test_role"
  path               = "/"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}
resource "aws_iam_role_policy" "modify_routes" {
  name   = "awsnycast_modify_routes"
  role   = aws_iam_role.role.id
  policy = data.aws_iam_policy_document.modify_routes_policy.json
}
resource "aws_iam_instance_profile" "test_profile" {
  name = "awsnycast_test_profile"
  role = aws_iam_role.role.name
}
