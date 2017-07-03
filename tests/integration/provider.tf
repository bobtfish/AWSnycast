provider "aws" {
    region = "eu-west-1"
    shared_credentials_file = "${pathexpand("~/.aws/credentials")}"
    profile = "default"
}

