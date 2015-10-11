resource "aws_iam_instance_profile" "test_profile" {
    name = "test_profile"
    roles = ["${aws_iam_role.role.name}"]
}

resource "aws_iam_role" "role" {
    name = "test_role"
    path = "/"
    assume_role_policy = <<EOF
{
    "Version": "2008-10-17",
    "Statement": [
        {
            "Action": "sts:AssumeRole",
            "Principal": {"AWS": "*"},
            "Effect": "Allow",
            "Sid": ""
        }
    ]
}
EOF
}

