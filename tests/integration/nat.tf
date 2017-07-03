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
    user_data = "${replace(file("${path.module}/nat.conf"), "__MYAZ__", "eu-west-1a")}"
    iam_instance_profile = "${aws_iam_instance_profile.test_profile.id}"
    provisioner "remote-exec" {
        inline = [
          "while sudo pkill -0 cloud-init; do sleep 2; done"
        ]
        connection {
          user = "ubuntu"
          private_key = "${file("id_rsa")}"
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
    user_data = "${replace(file("${path.module}/nat.conf"), "__MYAZ__", "eu-west-1b")}"
    iam_instance_profile = "${aws_iam_instance_profile.test_profile.id}"
    provisioner "remote-exec" {
        inline = [
          "while sudo pkill -0 cloud-init; do sleep 2; done"
        ]
        connection {
          user = "ubuntu"
          private_key = "${file("id_rsa")}"
        }
    }
}

output "nat_public_ips" {
    value = "${aws_instance.nat-a.public_ip},${aws_instance.nat-b.public_ip}"
}

