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

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "1.37.0"

  name = "test-ecs-instance-ready"
  cidr = "10.0.0.0/16"

  azs            = ["us-west-2a"]
  public_subnets = ["10.0.0.0/24"]

  tags = {
    Test = "ecs-instance-ready"
  }
}

module "security_group" {
  source  = "terraform-aws-modules/security-group/aws//modules/ssh"
  version = "2.1.0"

  name   = "test-ecs-instance-ready-ssh"
  vpc_id = "${module.vpc.vpc_id}"

  ingress_cidr_blocks = ["0.0.0.0/0"]
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
  name = "test-ecs-instance-ready-instance-role"

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
  name = "ecs-instance"
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
  name = "test-ecs-instance-ready"
  role = "${aws_iam_role.ecs_instance.id}"
}

resource "aws_launch_configuration" "ecs_instance" {
  name     = "test-ecs-instance-ready"
  image_id = "${data.aws_ami.amazon_linux.image_id}"

  iam_instance_profile        = "${aws_iam_instance_profile.ecs_instance.id}"
  security_groups             = ["${module.security_group.this_security_group_id}"]
  instance_type               = "t2.micro"
  associate_public_ip_address = false

  user_data = <<EOF
#!/bin/bash
echo ECS_CLUSTER=${aws_ecs_cluster.test.name} >> /etc/ecs/ecs.config
EOF
}

module "asg" {
  source  = "terraform-aws-modules/autoscaling/aws"
  version = "2.8.0"

  name                 = "test-ecs-instance-ready"
  health_check_type    = "EC2"
  vpc_zone_identifier  = ["${module.vpc.public_subnets}"]
  create_lc            = false
  launch_configuration = "${aws_launch_configuration.ecs_instance.id}"

  desired_capacity = 0
  min_size         = 0
  max_size         = 1

  wait_for_capacity_timeout = 0
}

resource "aws_ecs_cluster" "test" {
  name = "test-ecs-instance-ready"
}

resource "aws_ecs_task_definition" "test" {
  family = "test"

  container_definitions = <<EOF
[
  {
    "name": "webserver",
    "image": "nginx:1.14.0-alpine",
    "cpu": 512,
    "essential": true,
    "memory": 256
  }
]
EOF
}

resource "aws_ecs_service" "test" {
  name                               = "test"
  cluster                            = "${aws_ecs_cluster.test.id}"
  task_definition                    = "${aws_ecs_task_definition.test.arn}"
  desired_count                      = 1
  deployment_maximum_percent         = 100
  deployment_minimum_healthy_percent = 50
}

module "ready" {
  source = "../../terraform/ecs_instance_ready"

  lambda_version = "${var.lambda_version}"

  autoscaling_group_name = "${module.asg.this_autoscaling_group_name}"
  ecs_cluster_name       = "${aws_ecs_cluster.test.name}"
  required_task_families = ["${var.required_task_families}"]
  timeout                = "${var.timeout}"
}

output "start_poller_lambda_arn" {
  value = "${module.ready.start_poller_lambda_arn}"
}

output "step_function_arn" {
  value = "${module.ready.step_function_arn}"
}

output "autoscaling_group_name" {
  value = "${module.asg.this_autoscaling_group_name}"
}

output "ecs_cluster_name" {
  value = "${aws_ecs_cluster.test.name}"
}
