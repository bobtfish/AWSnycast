module "ami" {
  source = "github.com/terraform-community-modules/tf_aws_ubuntu_ami/ebs"
  instance_type = "m3.medium"
  region = "eu-west-1"
  distribution = "vivid"
}

