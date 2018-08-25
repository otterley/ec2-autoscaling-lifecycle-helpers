resource "aws_lambda_function" "start_drainer" {
  function_name = "${format("%.64s", "start-ecs-inst-drain-${var.autoscaling_group_name}")}"
  description   = "Start ECS instance drainer for ${var.autoscaling_group_name} group"
  role          = "${aws_iam_role.start_drainer.arn}"

  s3_bucket = "${var.s3_bucket}"
  s3_key    = "${var.lambda_version}/start-ecs-instance-drainer.zip"
  handler   = "start-ecs-instance-drainer"
  runtime   = "go1.x"

  environment {
    variables = {
      STATE_MACHINE_ARN          = "${aws_sfn_state_machine.drainer.id}"
      ECS_CLUSTER                = "${coalesce(var.ecs_cluster_name, var.autoscaling_group_name)}"
      TIMEOUT                    = "${var.timeout}"
      STOP_ALL_NON_SERVICE_TASKS = "${var.stop_all_non_service_tasks}"
      STOP_TASK_GROUPS           = "${join(",", var.stop_task_groups)}"
    }
  }
}

data "aws_iam_policy_document" "start_drainer_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "start_drainer_policy" {
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
      "ecs:UpdateContainerInstancesState",
      "ecs:DescribeContainerInstances",
      "ecs:ListTasks",
      "ecs:DescribeTasks",
      "ecs:StopTask",
    ]

    resources = ["*"]
  }

  statement {
    actions = [
      "ecs:ListContainerInstances",
    ]

    resources = ["*"]
  }

  statement {
    actions   = ["states:StartExecution"]
    resources = ["${aws_sfn_state_machine.drainer.id}"]
  }
}

resource "aws_iam_role" "start_drainer" {
  name               = "${format("%.64s", "start-drainer-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.start_drainer_assume_role.json}"
}

resource "aws_iam_role_policy" "start_drainer" {
  name   = "start-drainer"
  role   = "${aws_iam_role.start_drainer.name}"
  policy = "${data.aws_iam_policy_document.start_drainer_policy.json}"
}
