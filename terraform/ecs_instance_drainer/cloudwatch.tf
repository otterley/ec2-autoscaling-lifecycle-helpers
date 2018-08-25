resource "aws_cloudwatch_event_rule" "terminating" {
  name        = "${format("%.64s", "ecs_inst_drain-${var.autoscaling_group_name}")}"
  description = "Drain ECS instance for ${var.autoscaling_group_name}"

  event_pattern = <<PATTERN
{
    "source": "aws.autoscaling",
    "detail-type": [ "EC2 Instance-terminate Lifecycle Action" ],
    "detail": {
        "AutoScalingGroupName": [ "${var.autoscaling_group_name}" ]
   }
}
PATTERN
}

resource "aws_cloudwatch_event_target" "terminating" {
  rule = "${aws_cloudwatch_event_rule.terminating.name}"
  arn  = "${aws_lambda_function.start_drainer.arn}"
}

resource "aws_lambda_permission" "start_drainer" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.start_drainer.function_name}"
  principal     = "events.amazonaws.com"
  source_arn    = "${aws_cloudwatch_event_rule.terminating.arn}"
}
