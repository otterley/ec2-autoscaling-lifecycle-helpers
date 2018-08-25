resource "aws_autoscaling_lifecycle_hook" "terminate" {
  name                   = "ecs_instance_drainer"
  autoscaling_group_name = "${var.autoscaling_group_name}"
  default_result         = "CONTINUE"
  heartbeat_timeout      = "${var.wait_interval * 2}"
  lifecycle_transition   = "autoscaling:EC2_INSTANCE_TERMINATING"
}
