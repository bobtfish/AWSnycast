resource "aws_vpc" "main" {
    cidr_block = "10.0.0.0/16"
    enable_dns_support = true
    enable_dns_hostnames = true
    tags {
        Name = "main vpc"
    }
}

resource "aws_internet_gateway" "gw" {
    vpc_id = "${aws_vpc.main.id}"

    tags {
        Name = "main igw"
    }
}

resource "aws_route_table" "public" {
    vpc_id = "${aws_vpc.main.id}"
    route {
        cidr_block = "0.0.0.0/0"
        gateway_id = "${aws_internet_gateway.gw.id}"
    }

    tags {
        Name = "public"
    }
}

resource "aws_route_table" "privatea" {
    vpc_id = "${aws_vpc.main.id}"
    route { 
        cidr_block = "0.0.0.0/0"
        instance_id = "${aws_instance.nat-a.id}"
    }
    route {
        cidr_block = "192.168.1.1/32"
        instance_id = "${aws_instance.nat-a.id}"
    }
    tags {
        Name = "private a"
        az = "eu-west-1a"
        type = "private"
    }
}

resource "aws_route_table" "privateb" {
    vpc_id = "${aws_vpc.main.id}"
    route {
        cidr_block = "0.0.0.0/0"
        instance_id = "${aws_instance.nat-b.id}"
    }
    route {
        cidr_block = "192.168.1.1/32"
        instance_id = "${aws_instance.nat-b.id}"
    }
    tags {
        Name = "private b"
        az = "eu-west-1b"
        type = "private"
    }
}

resource "aws_subnet" "publica" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.0.0/24"
    map_public_ip_on_launch = true
    availability_zone = "eu-west-1a"

    tags {
        Name = "eu-west-1a public"
    }
}

resource "aws_subnet" "publicb" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.1.0/24"
    map_public_ip_on_launch = true
    availability_zone = "eu-west-1b"

    tags {
        Name = "eu-west-1b public"
    }
}

resource "aws_route_table_association" "publica" {
    subnet_id = "${aws_subnet.publica.id}"
    route_table_id = "${aws_route_table.public.id}"
}

resource "aws_route_table_association" "publicb" {
    subnet_id = "${aws_subnet.publicb.id}"
    route_table_id = "${aws_route_table.public.id}"
}

resource "aws_subnet" "privatea" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.10.0/24"
    availability_zone = "eu-west-1a"

    tags {
        Name = "eu-west-1a private"
    }
}

resource "aws_subnet" "privateb" {
    vpc_id = "${aws_vpc.main.id}"
    cidr_block = "10.0.11.0/24"
    availability_zone = "eu-west-1b"

    tags {
        Name = "eu-west-1b private"
    }
}

resource "aws_route_table_association" "privatea" {
    subnet_id = "${aws_subnet.privatea.id}"
    route_table_id = "${aws_route_table.privatea.id}"
}

resource "aws_route_table_association" "privateb" {
    subnet_id = "${aws_subnet.privateb.id}"
    route_table_id = "${aws_route_table.privateb.id}"
}

