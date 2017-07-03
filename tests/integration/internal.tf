resource "aws_instance" "internal-a" {
    ami = "${module.ami.ami_id}"
    instance_type = "m3.medium"
    key_name = "${aws_key_pair.awsnycast.key_name}"
    subnet_id = "${aws_subnet.privatea.id}"
    security_groups = ["${aws_security_group.allow_all.id}"]
    tags {
        Name = "internal eu-west-1a"
    }
    user_data = "${replace(file("${path.module}/internal.conf"), "__NETWORKPREFIX__", "10.0")}"
    provisioner "remote-exec" {
        inline = [
          "while sudo pkill -0 cloud-init; do sleep 2; done"
        ]
        connection {
          user = "ubuntu"
          private_key = "${file("id_rsa")}"
          bastion_host = "${aws_instance.nat-a.public_ip}"
        }
    }
}

resource "aws_instance" "internal-b" {
    ami = "${module.ami.ami_id}"
    instance_type = "m3.medium"
    key_name = "${aws_key_pair.awsnycast.key_name}"
    subnet_id = "${aws_subnet.privateb.id}"
    security_groups = ["${aws_security_group.allow_all.id}"]
    tags {
        Name = "internal eu-west-1b"
    }
    user_data = "${replace(file("${path.module}/internal.conf"), "__NETWORKPREFIX__", "10.0")}"
    provisioner "remote-exec" {
        inline = [
          "while sudo pkill -0 cloud-init; do sleep 2; done"
        ]
        connection {
          user = "ubuntu"
          private_key = "${file("id_rsa")}"
          bastion_host = "${aws_instance.nat-b.public_ip}"
        }
    }
}

output "internal_private_ips" {
    value = "${aws_instance.internal-a.private_ip},${aws_instance.internal-b.private_ip}"
}

