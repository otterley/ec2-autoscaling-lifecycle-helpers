resource "aws_cloudwatch_event_rule" "launching" {
  name        = "${format("%.64s", "kafka_ready-${var.autoscaling_group_name}")}"
  description = "Launch ECS instance for ${var.autoscaling_group_name}"

  event_pattern = <<PATTERN
{
    "detail-type": [ "EC2 Instance-launch Lifecycle Action" ],
    "detail": {
        "AutoScalingGroupName": [ "${var.autoscaling_group_name}" ]
   }
}
PATTERN
}

resource "aws_cloudwatch_event_target" "launching" {
  rule = "${aws_cloudwatch_event_rule.launching.name}"
  arn  = "${aws_lambda_function.start_poller.arn}"
}

resource "aws_lambda_permission" "start_poller" {
  statement_id  = "AllowExecutionFromCloudWatch"
  action        = "lambda:InvokeFunction"
  function_name = "${aws_lambda_function.start_poller.function_name}"
  principal     = "events.amazonaws.com"
  source_arn    = "${aws_cloudwatch_event_rule.launching.arn}"
}
