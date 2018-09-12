provider "aws" {}

variable "lambda_version" {
  type = "string"
}

variable "timeout" {
  default = "5m"
}

variable "required_task_families" {
  default = ["test"]
}

locals {
  azs = ["us-west-2a", "us-west-2b", "us-west-2c"]
}

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "1.37.0"

  name = "test-kafka-ready"
  cidr = "10.0.0.0/16"

  azs                = "${local.azs}"
  public_subnets     = ["10.0.0.0/24", "10.0.1.0/24", "10.0.2.0/24"]
  private_subnets    = ["10.0.3.0/24", "10.0.4.0/24", "10.0.5.0/24"]
  enable_nat_gateway = true

  tags = {
    Test = "kafka-ready"
  }
}

resource "aws_security_group" "bastion" {
  name   = "bastion"
  vpc_id = "${module.vpc.vpc_id}"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "test" {
  name   = "test-kafka-ready"
  vpc_id = "${module.vpc.vpc_id}"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = 9092
    to_port   = 9092
    protocol  = "tcp"
    self      = true
  }

  ingress {
    from_port = 2181
    to_port   = 2181
    protocol  = "tcp"
    self      = true
  }

  ingress {
    from_port = 2888
    to_port   = 2888
    protocol  = "tcp"
    self      = true
  }

  ingress {
    from_port = 3888
    to_port   = 3888
    protocol  = "tcp"
    self      = true
  }

  ingress {
    from_port       = 22
    to_port         = 22
    protocol        = "tcp"
    security_groups = ["${aws_security_group.bastion.id}"]
  }
}

data "aws_ami" "amazon_linux" {
  most_recent = true

  filter {
    name   = "name"
    values = ["amzn-ami-*-amazon-ecs-optimized"]
  }

  filter {
    name   = "owner-alias"
    values = ["amazon"]
  }
}

resource "aws_iam_role" "ecs_instance" {
  name = "test-kafka-ready-instance-role"

  assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "ecs_instance" {
  name = "kafka"
  role = "${aws_iam_role.ecs_instance.id}"

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:CreateCluster",
        "ecs:DeregisterContainerInstance",
        "ecs:DiscoverPollEndpoint",
        "ecs:Poll",
        "ecs:RegisterContainerInstance",
        "ecs:StartTelemetrySession",
        "ecs:Submit*",
        "ecr:GetAuthorizationToken",
        "ecr:BatchCheckLayerAvailability",
        "ecr:GetDownloadUrlForLayer",
        "ecr:BatchGetImage",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}

resource "aws_iam_instance_profile" "ecs_instance" {
  name = "test-kafka-ready"
  role = "${aws_iam_role.ecs_instance.id}"
}

resource "aws_launch_configuration" "kafka" {
  name_prefix = "test-kafka-ready"
  image_id    = "${data.aws_ami.amazon_linux.image_id}"

  iam_instance_profile        = "${aws_iam_instance_profile.ecs_instance.id}"
  security_groups             = ["${aws_security_group.test.id}"]
  instance_type               = "t2.medium"
  associate_public_ip_address = false

  key_name = "michael@dynamine.net"

  user_data = <<EOF
#!/bin/bash
echo ECS_CLUSTER=${aws_ecs_cluster.kafka.name} >> /etc/ecs/ecs.config
EOF

  lifecycle {
    create_before_destroy = true
  }
}

module "kafka_asg" {
  source  = "terraform-aws-modules/autoscaling/aws"
  version = "2.8.0"

  name                 = "test-kafka-ready"
  health_check_type    = "EC2"
  vpc_zone_identifier  = ["${module.vpc.private_subnets}"]
  create_lc            = false
  launch_configuration = "${aws_launch_configuration.kafka.id}"

  desired_capacity = 3
  min_size         = 2
  max_size         = 3

  wait_for_capacity_timeout = 0
}

resource "aws_instance" "bastion" {
  ami                         = "${data.aws_ami.amazon_linux.image_id}"
  instance_type               = "t2.micro"
  vpc_security_group_ids      = ["${aws_security_group.bastion.id}"]
  subnet_id                   = "${element(module.vpc.public_subnets, count.index)}"
  associate_public_ip_address = true

  key_name = "michael@dynamine.net"

  tags = {
    Name = "bastion"
  }
}

resource "aws_instance" "zookeeper" {
  count                       = 3
  ami                         = "${data.aws_ami.amazon_linux.image_id}"
  instance_type               = "t2.small"
  vpc_security_group_ids      = ["${aws_security_group.test.id}"]
  subnet_id                   = "${element(module.vpc.private_subnets, count.index)}"
  iam_instance_profile        = "${aws_iam_instance_profile.ecs_instance.id}"
  associate_public_ip_address = false

  key_name = "michael@dynamine.net"

  tags = {
    Name = "test-kafka-ready-zk-${count.index}"
  }

  user_data = <<EOF
#!/bin/bash
echo ECS_CLUSTER=${element(aws_ecs_cluster.zookeeper.*.name, count.index)} >> /etc/ecs/ecs.config
EOF
}

resource "aws_ecs_cluster" "kafka" {
  name = "test-kafka-ready"
}

resource "aws_ecs_cluster" "zookeeper" {
  count = 3
  name  = "test-kafka-ready-zk-${count.index}"
}

resource "aws_ecs_task_definition" "kafka" {
  family       = "kafka"
  network_mode = "host"

  container_definitions = <<EOF
[
  {
    "name": "kafka",
    "image": "wurstmeister/kafka:2.11-1.1.1",
    "cpu": 1024,
    "essential": true,
    "memory": 1024,
    "environment": [
      {
        "name": "KAFKA_ZOOKEEPER_CONNECT",
        "value": "${aws_network_interface.zookeeper.0.private_ips[0]}:2181,${aws_network_interface.zookeeper.1.private_ips[0]}:2181,${aws_network_interface.zookeeper.2.private_ips[0]}:2181"
      },
      {
        "name": "HOSTNAME_COMMAND",
        "value": "curl -sSf http://169.254.169.254/latest/meta-data/hostname"
      },
      {
        "name": "RACK_COMMAND",
        "value": "curl -sSf http://169.254.169.254/latest/meta-data/placement/availability-zone"
      },
      {
        "name": "KAFKA_CREATE_TOPICS",
        "value": "test:1:3"
      }
    ]
  }
]
EOF
}

resource "aws_ecs_task_definition" "zookeeper" {
  count        = 3
  family       = "zookeeper-${count.index}"
  network_mode = "host"

  container_definitions = <<EOF
[
  {
    "name": "kafka",
    "image": "zookeeper:3.4.13",
    "cpu": 256,
    "essential": true,
    "memory": 512,
    "environment": [
      {
        "name": "ZOO_MY_ID",
        "value": "${count.index}"
      },
      {
        "name": "ZOO_SERVERS",
        "value": "server.0=${aws_network_interface.zookeeper.0.private_ips[0]}:2888:3888 server.1=${aws_network_interface.zookeeper.1.private_ips[0]}:2888:3888 server.2=${aws_network_interface.zookeeper.2.private_ips[0]}:2888:3888"
      }
    ]
  }
]
EOF
}

resource "aws_ecs_service" "kafka" {
  name                               = "kafka"
  cluster                            = "${aws_ecs_cluster.kafka.id}"
  task_definition                    = "${aws_ecs_task_definition.kafka.arn}"
  desired_count                      = 0
  deployment_maximum_percent         = 200
  deployment_minimum_healthy_percent = 100

  placement_constraints {
    type = "distinctInstance"
  }
}

resource "aws_ecs_service" "zookeeper" {
  name                               = "zookeeper"
  count                              = 3
  cluster                            = "${element(aws_ecs_cluster.zookeeper.*.id, count.index)}"
  task_definition                    = "${element(aws_ecs_task_definition.zookeeper.*.arn, count.index)}"
  desired_count                      = 1
  deployment_maximum_percent         = 200
  deployment_minimum_healthy_percent = 100

  placement_constraints {
    type = "distinctInstance"
  }
}

resource "aws_network_interface" "zookeeper" {
  count           = 3
  subnet_id       = "${element(module.vpc.private_subnets, count.index)}"
  security_groups = ["${aws_security_group.test.id}"]

  attachment {
    instance     = "${element(aws_instance.zookeeper.*.id, count.index)}"
    device_index = 1
  }
}

module "instance_ready" {
  source = "../../terraform/ecs_instance_ready"

  lambda_version         = "${var.lambda_version}"
  autoscaling_group_name = "${module.kafka_asg.this_autoscaling_group_name}"
  ecs_cluster_name       = "${aws_ecs_cluster.kafka.name}"
  required_task_families = ["${aws_ecs_task_definition.kafka.family}"]
  timeout                = "${var.timeout}"
}

module "kafka_ready" {
  source = "../../terraform/kafka_ready"

  lambda_version         = "${var.lambda_version}"
  autoscaling_group_name = "${module.kafka_asg.this_autoscaling_group_name}"
  subnet_ids             = ["${module.vpc.private_subnets}"]
}

# output "start_poller_lambda_arn" {
#   value = "${module.ready.start_poller_lambda_arn}"
# }


# output "step_function_arn" {
#   value = "${module.ready.step_function_arn}"
# }


# output "autoscaling_group_name" {
#   value = "${module.asg.this_autoscaling_group_name}"
# }


# output "ecs_cluster_name" {
#   value = "${aws_ecs_cluster.test.name}"
# }

