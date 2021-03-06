variable "autoscaling_group_name" {
  description = "Name of Auto Scaling Group to be managed"
  type        = "string"
}

variable "ecs_cluster_name" {
  description = "ECS cluster associated with Auto Scaling Group.  If blank, the value of autoscaling_group_name will be used."
  default     = ""
}

variable "wait_interval" {
  description = "Number of seconds to wait between poll attempts"
  default     = "30"
}

variable "timeout" {
  description = "Timeout after which instance will be terminated if not ready, as a Go duration string"
  default     = "5m"
}

variable "required_task_families" {
  description = "List of ECS task families that must also have at least 1 task running on instance"
  default     = []
}

variable "lambda_version" {
  type        = "string"
  description = "Lambda function version"
}

variable "s3_bucket" {
  description = "S3 bucket in which Lambda functions live"
  default     = "ec2-instance-lifecycle"
}
