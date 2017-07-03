resource "aws_iam_instance_profile" "test_profile" {
    name = "test_profile"
    role = "${aws_iam_role.role.name}"
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

resource "aws_iam_role_policy" "modify_routes" {
    name = "modify_routes"
    role = "${aws_iam_role.role.id}"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
        {
            "Action": [
                "ec2:ReplaceRoute",
                "ec2:CreateRoute",
                "ec2:DeleteRoute",
                "ec2:DescribeRouteTables",
                "ec2:DescribeNetworkInterfaces",
                "ec2:DescribeInstanceAttribute"
            ],
            "Effect": "Allow",
            "Resource": "*"
        }
    ]
}
EOF
}

