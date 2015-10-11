module "ami" {
  source = "github.com/terraform-community-modules/tf_aws_ubuntu_ami/ebs"
  instance_type = "m3.medium"
  region = "eu-west-1"
  distribution = "trusty"
}

resource "aws_instance" "nat" {
    ami = "${module.ami.ami_id}"
    instance_type = "m3.medium"
    source_dest_check = false
    key_name = "${aws_key_pair.awsnycast.key_name}"
    subnet_id = "${aws_subnet.publica.id}"
    security_groups = ["${aws_security_group.allow_all.id}"]
    tags {
        Name = "nat eu-west-1a"
    }
    user_data = "${replace(file(\"${path.module}/nat.conf\"), \"__NETWORKPREFIX__\", \"10.0\")}"
    provisioner "remote-exec" {
        inline = [
          "while sudo pkill -0 cloud-init; do sleep 2; done"
        ]
        connection {
          user = "ubuntu"
          key_file = "id_rsa"
        }
    }
}

output "nat_public_ips" {
    value = "${aws_instance.nat.public_ip}"
}

