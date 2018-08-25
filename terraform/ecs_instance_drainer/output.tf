output "start_drainer_lambda_arn" {
  value = "${aws_lambda_function.start_drainer.arn}"
}

output "step_function_arn" {
  value = "${aws_sfn_state_machine.drainer.id}"
}
