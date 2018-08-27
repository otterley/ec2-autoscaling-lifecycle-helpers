resource "aws_autoscaling_lifecycle_hook" "ready" {
  name                   = "ecs_instance_ready"
  autoscaling_group_name = "${var.autoscaling_group_name}"
  default_result         = "ABANDON"
  heartbeat_timeout      = "${var.wait_interval * 2}"
  lifecycle_transition   = "autoscaling:EC2_INSTANCE_LAUNCHING"
}
