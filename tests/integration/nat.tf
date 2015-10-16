module "ami" {
  source = "github.com/terraform-community-modules/tf_aws_ubuntu_ami/ebs"
  instance_type = "m3.medium"
  region = "eu-west-1"
  distribution = "vivid"
}

resource "aws_instance" "nat-a" {
    ami = "${module.ami.ami_id}"
    instance_type = "m3.medium"
    source_dest_check = false
    key_name = "${aws_key_pair.awsnycast.key_name}"
    subnet_id = "${aws_subnet.publica.id}"
    security_groups = ["${aws_security_group.allow_all.id}"]
    tags {
        Name = "nat eu-west-1a"
    }
    user_data = "${replace(replace(replace(file(\"${path.module}/nat.conf\"), \"__NETWORKPREFIX__\", \"10.0\"), \"__A_EXTRA__\", \"\"), \"__B_EXTRA__\", \"if_unhealthy: true\")}"
    iam_instance_profile = "${aws_iam_instance_profile.test_profile.id}"
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

resource "aws_instance" "nat-b" {
    ami = "${module.ami.ami_id}"
    instance_type = "m3.medium"
    source_dest_check = false
    key_name = "${aws_key_pair.awsnycast.key_name}"
    subnet_id = "${aws_subnet.publicb.id}"
    security_groups = ["${aws_security_group.allow_all.id}"]
    tags {
        Name = "nat eu-west-1b"
    }
    user_data = "${replace(replace(replace(file(\"${path.module}/nat.conf\"), \"__NETWORKPREFIX__\", \"10.0\"), \"__B_EXTRA__\", \"\"), \"__A_EXTRA__\", \"if_unhealthy: true\")}"
    iam_instance_profile = "${aws_iam_instance_profile.test_profile.id}"
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
    value = "${aws_instance.nat-a.public_ip},${aws_instance.nat-b.public_ip}"
}

