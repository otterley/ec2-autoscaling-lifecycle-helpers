variable "autoscaling_group_name" {
  description = "Name of Auto Scaling Group to be managed"
  type        = "string"
}

variable "ecs_cluster_name" {
  description = "ECS cluster associated with Auto Scaling Group.  If blank, the value of autoscaling_group_name will be used."
  default     = ""
}

variable "wait_interval" {
  description = "Number of seconds to wait between counting ECS tasks"
  default     = "30"
}

variable "timeout" {
  description = "Timeout after which instance will be terminated even if not drained, as a Go duration string"
  default     = "5m"
}

variable "stop_all_non_service_tasks" {
  description = "If true, stop all non-service tasks immediately"
  default     = "true"
}

variable "stop_task_groups" {
  description = "List of ECS task groups to stop immediately"
  default     = []
}

variable "lambda_version" {
  description = "Lambda function version"
}

variable "s3_bucket" {
  description = "S3 bucket in which Lambda functions live"
  default     = "ec2-instance-lifecycle"
}
