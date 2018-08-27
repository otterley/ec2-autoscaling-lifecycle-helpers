resource "aws_lambda_function" "check_deadline" {
  function_name = "${format("%.64s", "ecs-inst-rdy-chk-dead-${var.autoscaling_group_name}")}"
  description   = "ECS instance readiness poller - check_deadline for ${var.autoscaling_group_name} group"
  role          = "${aws_iam_role.check_deadline.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/check-deadline.zip"
  handler   = "check-deadline"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "check_deadline_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "check_deadline_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_role" "check_deadline" {
  name               = "${format("%.64s", "ecs-inst-rdy-chk-dead-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.check_deadline_assume_role.json}"
}

resource "aws_iam_role_policy" "check_deadline" {
  name   = "check-deadline"
  role   = "${aws_iam_role.check_deadline.name}"
  policy = "${data.aws_iam_policy_document.check_deadline_policy.json}"
}
