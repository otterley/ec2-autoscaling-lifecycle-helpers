resource "aws_lambda_function" "check_instance_ready" {
  function_name = "${format("%.64s", "ecs-check-inst-rdy-${var.autoscaling_group_name}")}"
  description   = "ECS instance poller - check-instance-readya for ${var.autoscaling_group_name} Auto Scaling Group"
  role          = "${aws_iam_role.check_instance_ready.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/check-ecs-instance-ready.zip"
  handler   = "check-ecs-instance-ready"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "check_instance_ready_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "check_instance_ready_policy" {
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
      "ecs:DescribeContainerInstances",
      "ecs:ListContainerInstances",
      "ecs:ListTasks",
    ]

    resources = ["*"]
  }
}

resource "aws_iam_role" "check_instance_ready" {
  name               = "${format("%.64s", "ecs-check-inst-rdy-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.check_instance_ready_assume_role.json}"
}

resource "aws_iam_role_policy" "check_instance_ready" {
  name   = "check_instance_ready"
  role   = "${aws_iam_role.check_instance_ready.name}"
  policy = "${data.aws_iam_policy_document.check_instance_ready_policy.json}"
}
