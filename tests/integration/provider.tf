provider "aws" {
  region = "us-west-2"
  default_tags {
    tags = {
      integration_test = "true"
      temporary        = "true"
      project          = "awsnycast"
    }
  }
}

provider "archive" {

}
