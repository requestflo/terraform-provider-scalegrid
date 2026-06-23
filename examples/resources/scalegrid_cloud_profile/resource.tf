# An AWS (EC2/VPC) cloud profile for MongoDB deployments.
resource "scalegrid_cloud_profile" "aws" {
  database            = "mongodb"
  name                = "aws-use1-a"
  region              = "us-east-1"
  access_key          = var.aws_access_key
  secret_key          = var.aws_secret_key
  vpc_id              = "vpc-0123456789abcdef0"
  subnet_id           = "subnet-0123456789abcdef0"
  vpc_cidr            = "10.0.0.0/16"
  subnet_cidr         = "10.0.1.0/24"
  security_group_id   = "sg-0123456789abcdef0"
  security_group_name = "scalegrid-mongo"
  connectivity_config = "INTERNET"
}
