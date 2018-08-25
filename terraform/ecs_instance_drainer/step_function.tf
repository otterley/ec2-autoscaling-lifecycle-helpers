resource "aws_sfn_state_machine" "drainer" {
  name     = "ecs_instance_drainer-${var.autoscaling_group_name}"
  role_arn = "${aws_iam_role.drainer.arn}"

  definition = <<EOF
{
    "Comment": "ECS instance drainer - ${var.autoscaling_group_name}",
    "StartAt": "CountRunningExecutions",
    "States": {
        "CountRunningExecutions": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.count_running_executions.arn}",
            "Next": "HaltIfRunningExecutions"
        },
        "HaltIfRunningExecutions": {
            "Type": "Choice",
            "Choices": [
                {
                    "Variable": "$.RunningExecutionCount",
                    "NumericGreaterThan": 1,
                    "Next": "AlreadyRunning"
                }
            ],
            "Default": "CountRunningTasks"
        },
        "AlreadyRunning": {
            "Type": "Fail",
            "Cause": "Another execution is already running"
        },
        "CheckDeadline": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.check_deadline.arn}",
            "Next": "HaltIfPastDeadline"
        },
        "HaltIfPastDeadline": {
            "Type": "Choice",
            "Choices": [
                {
                    "Variable": "$.PastDeadline",
                    "BooleanEquals": true,
                    "Next": "AbandonLifecycleAction"
                }
            ],
            "Default": "CountRunningTasks"
        },
        "CountRunningTasks": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.count_ecs_tasks.arn}",
            "Next": "CompleteIfNoTasks" 
        },
        "CompleteIfNoTasks": {
            "Type": "Choice",
            "Choices": [
                {
                    "Variable": "$.ECSTaskCount",
                    "NumericEquals": 0,
                    "Next": "ContinueLifecycleAction"
                }
            ],
            "Default": "Heartbeat"
        },
        "Heartbeat": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.record_lifecycle_heartbeat.arn}",
            "Next": "WaitAndCountAgain"
        },
        "WaitAndCountAgain": {
            "Type": "Wait",
            "Seconds": ${var.wait_interval},
            "Next": "CheckDeadline"
        },
        "ContinueLifecycleAction": {
            "Type": "Pass",
            "Result": {
                "LifecycleActionResult": "CONTINUE"
            },
            "ResultPath": "$.Params",
            "Next": "CompleteLifecycleAction"
        },
        "AbandonLifecycleAction": {
            "Type": "Pass",
            "Result": {
                "LifecycleActionResult": "ABANDON"
            },
            "ResultPath": "$.Params",
            "Next": "CompleteLifecycleAction"
        },
        "CompleteLifecycleAction": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.complete_lifecycle_action.arn}",
            "End": true
        }
    }
}
EOF
}

data "aws_iam_policy_document" "drainer_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "drainer_policy" {
  statement {
    actions = ["lambda:InvokeFunction"]

    resources = [
      "${aws_lambda_function.count_running_executions.arn}",
      "${aws_lambda_function.check_deadline.arn}",
      "${aws_lambda_function.count_ecs_tasks.arn}",
      "${aws_lambda_function.complete_lifecycle_action.arn}",
      "${aws_lambda_function.record_lifecycle_heartbeat.arn}",
    ]
  }
}

resource "aws_iam_role" "drainer" {
  name               = "${format("%.64s", "ecs-instance-drainer-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.drainer_assume_role.json}"
}

resource "aws_iam_role_policy" "drainer" {
  name   = "ecs-instance-drainer"
  role   = "${aws_iam_role.drainer.name}"
  policy = "${data.aws_iam_policy_document.drainer_policy.json}"
}
