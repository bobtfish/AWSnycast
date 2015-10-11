resource "aws_key_pair" "awsnycast" {
  key_name = "AWSNycast-key"
  public_key = "${var.deploy_ssh_pubkey}"
}

