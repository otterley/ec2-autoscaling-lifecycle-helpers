resource "aws_sfn_state_machine" "poller" {
  name     = "ecs_instance_ready_poller-${var.autoscaling_group_name}"
  role_arn = "${aws_iam_role.poller.arn}"

  definition = <<EOF
{
    "Comment": "ECS instance readiness poller - ${var.autoscaling_group_name}",
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
            "Default": "CheckReady"
        },
        "CheckReady": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.check_instance_ready.arn}",
            "Next": "CompleteIfInstanceReady" 
        },
        "CompleteIfInstanceReady": {
            "Type": "Choice",
            "Choices": [
                {
                    "Variable": "$.Ready",
                    "BooleanEquals": true,
                    "Next": "ContinueLifecycleAction"
                }
            ],
            "Default": "Heartbeat"
        },
        "Heartbeat": {
            "Type": "Task",
            "Resource": "${aws_lambda_function.record_lifecycle_heartbeat.arn}",
            "Next": "WaitAndCheckAgain"
        },
        "WaitAndCheckAgain": {
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

data "aws_iam_policy_document" "poller_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["states.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "poller_policy" {
  statement {
    actions = ["lambda:InvokeFunction"]

    resources = [
      "${aws_lambda_function.count_running_executions.arn}",
      "${aws_lambda_function.check_deadline.arn}",
      "${aws_lambda_function.check_instance_ready.arn}",
      "${aws_lambda_function.complete_lifecycle_action.arn}",
      "${aws_lambda_function.record_lifecycle_heartbeat.arn}",
    ]
  }
}

resource "aws_iam_role" "poller" {
  name               = "${format("%.64s", "ecs-inst-rdy-poller-${var.autoscaling_group_name}")}"
  assume_role_policy = "${data.aws_iam_policy_document.poller_assume_role.json}"
}

resource "aws_iam_role_policy" "poller" {
  name   = "ecs-instance-poller"
  role   = "${aws_iam_role.poller.name}"
  policy = "${data.aws_iam_policy_document.poller_policy.json}"
}
