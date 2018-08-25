resource "aws_lambda_function" "count_ecs_tasks" {
  function_name = "${format("%.64s", "ecs-inst-drain-count-tasks-${var.autoscaling_group_name}")}"
  description   = "ECS instance drainer - count-ecs-tasks for ${var.autoscaling_group_name} Auto Scaling Group"
  role          = "${aws_iam_role.count_ecs_tasks.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/count-ecs-tasks.zip"
  handler   = "count-ecs-tasks"
  runtime   = "go1.x"
}

data "aws_iam_policy_document" "count_ecs_tasks_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "count_ecs_tasks_policy" {
  statement {
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["ecs:ListTasks"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "count_ecs_tasks" {
  name               = "${format("%.64s", "ecs-inst-drain-count-tasks-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.count_ecs_tasks_assume_role.json}"
}

resource "aws_iam_role_policy" "count_ecs_tasks" {
  name   = "count_ecs_tasks"
  role   = "${aws_iam_role.count_ecs_tasks.name}"
  policy = "${data.aws_iam_policy_document.count_ecs_tasks_policy.json}"
}
