output "start_poller_lambda_arn" {
  value = "${aws_lambda_function.start_poller.arn}"
}

output "step_function_arn" {
  value = "${aws_sfn_state_machine.poller.id}"
}
