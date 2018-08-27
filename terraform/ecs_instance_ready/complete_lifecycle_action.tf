resource "aws_lambda_function" "complete_lifecycle_action" {
  function_name = "${format("%.64s", "ecs-inst-rdy-lc-act-${var.autoscaling_group_name}")}"
  description   = "ECS instance readiness poller - complete_lifecycle_action for ${var.autoscaling_group_name} group"
  role          = "${aws_iam_role.complete_lifecycle_action.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/complete-lifecycle-action.zip"
  handler   = "complete-lifecycle-action"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "complete_lifecycle_action_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "complete_lifecycle_action_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["autoscaling:CompleteLifecycleAction"]
    resources = ["arn:aws:autoscaling:*:*:autoScalingGroup:*:autoScalingGroupName/${var.autoscaling_group_name}"]
  }
}

resource "aws_iam_role" "complete_lifecycle_action" {
  name               = "${format("%.64s", "ecs-inst-rdy-lc-act-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.complete_lifecycle_action_assume_role.json}"
}

resource "aws_iam_role_policy" "complete_lifecycle_action" {
  name   = "complete-lifecycle-action"
  role   = "${aws_iam_role.complete_lifecycle_action.name}"
  policy = "${data.aws_iam_policy_document.complete_lifecycle_action_policy.json}"
}
