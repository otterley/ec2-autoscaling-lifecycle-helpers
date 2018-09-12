resource "aws_lambda_function" "count_running_executions" {
  function_name = "${format("%.64s", "kafka-rdy_count_running-${var.autoscaling_group_name}")}"
  description   = "Kafka readiness poller - count-running-executions for ${var.autoscaling_group_name} Auto Scaling Group"
  role          = "${aws_iam_role.count_running_executions.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/count-running-executions.zip"
  handler   = "count-running-executions"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "count_running_executions_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "count_running_executions_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["states:ListExecutions"]
    resources = ["${aws_sfn_state_machine.poller.id}"]
  }
}

resource "aws_iam_role" "count_running_executions" {
  name               = "${format("%.64s", "kafka-rdy-count_running-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.count_running_executions_assume_role.json}"
}

resource "aws_iam_role_policy" "count_running_executions" {
  name   = "count_running_executions"
  role   = "${aws_iam_role.count_running_executions.name}"
  policy = "${data.aws_iam_policy_document.count_running_executions_policy.json}"
}
