resource "aws_lambda_function" "record_lifecycle_heartbeat" {
  function_name = "${format("%.64s", "ecs-inst-drain-lc-htbt-${var.autoscaling_group_name}")}"
  description   = "ECS instance drainer - record_lifecycle_heartbeat for ${var.autoscaling_group_name} group"
  role          = "${aws_iam_role.record_lifecycle_heartbeat.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/record-lifecycle-heartbeat.zip"
  handler   = "record-lifecycle-heartbeat"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "record_lifecycle_heartbeat_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "record_lifecycle_heartbeat_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["autoscaling:RecordLifecycleActionHeartbeat"]
    resources = ["arn:aws:autoscaling:*:*:autoScalingGroup:*:autoScalingGroupName/${var.autoscaling_group_name}"]
  }
}

resource "aws_iam_role" "record_lifecycle_heartbeat" {
  name               = "${format("%.64s", "ecs-inst-drain-lc-htbt-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.record_lifecycle_heartbeat_assume_role.json}"
}

resource "aws_iam_role_policy" "record_lifecycle_heartbeat" {
  name   = "record-lifecycle-heartbeat"
  role   = "${aws_iam_role.record_lifecycle_heartbeat.name}"
  policy = "${data.aws_iam_policy_document.record_lifecycle_heartbeat_policy.json}"
}
