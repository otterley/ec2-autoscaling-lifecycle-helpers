resource "aws_lambda_function" "check_kafka_ready" {
  function_name = "${format("%.64s", "kafka-rdy-${var.autoscaling_group_name}")}"
  description   = "Kafka readiness checker for ${var.autoscaling_group_name} Auto Scaling Group"
  role          = "${aws_iam_role.check_kafka_ready.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/check-kafka-ready.zip"
  handler   = "check-kafka-ready"
  runtime   = "go1.x"

  vpc_config {
    subnet_ids         = ["${var.subnet_ids}"]
    security_group_ids = ["${concat(list(aws_security_group.check_kafka_ready.id), var.security_group_ids)}"]
  }
}

data "aws_iam_policy_document" "check_kafka_ready_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "check_kafka_ready_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_role" "check_kafka_ready" {
  name               = "${format("%.64s", "kafka-rdy-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.check_kafka_ready_assume_role.json}"
}

resource "aws_iam_role_policy" "check_kafka_ready" {
  name   = "check_kafka_ready"
  role   = "${aws_iam_role.check_kafka_ready.name}"
  policy = "${data.aws_iam_policy_document.check_kafka_ready_policy.json}"
}

resource "aws_security_group" "check_kafka_ready" {
  name        = "${format("%.64s", "kafka-rdy-${var.autoscaling_group_name}")}"
  description = "Allow egress traffic for Kafka readiness Lambda function"

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
