resource "aws_lambda_function" "start_poller" {
  function_name = "${format("%.64s", "start-kafka-rdy-poller-${var.autoscaling_group_name}")}"
  description   = "Start ECS instance poller for ${var.autoscaling_group_name} group"
  role          = "${aws_iam_role.start_poller.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/start-kafka-ready-poller.zip"
  handler   = "start-kafka-ready-poller"
  runtime   = "go1.x"

  environment {
    variables = {
      STATE_MACHINE_ARN = "${aws_sfn_state_machine.poller.id}"
      TIMEOUT           = "${var.timeout}"
    }
  }
}

data "aws_iam_policy_document" "start_poller_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "start_poller_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }

  statement {
    actions = [
      "ec2:DescribeInstances",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["states:StartExecution"]
    resources = ["${aws_sfn_state_machine.poller.id}"]
  }
}

resource "aws_iam_role" "start_poller" {
  name               = "${format("%.64s", "start-kafka-rdy-poller-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.start_poller_assume_role.json}"
}

resource "aws_iam_role_policy" "start_poller" {
  name   = "start-poller"
  role   = "${aws_iam_role.start_poller.name}"
  policy = "${data.aws_iam_policy_document.start_poller_policy.json}"
}
