resource "aws_key_pair" "awsnycast" {
  key_name = "AWSnycast-key"
  public_key = "${var.deploy_ssh_pubkey}"
}

