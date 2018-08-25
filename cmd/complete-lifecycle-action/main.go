package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
)

func putLifecycleAction(request internal.DrainParameters) (internal.DrainParameters, error) {
	response := request
	client := autoscaling.New(session.Must(session.NewSession()))

	_, err := client.CompleteLifecycleAction(
		&autoscaling.CompleteLifecycleActionInput{
			AutoScalingGroupName:  aws.String(request.AutoScalingGroupName),
			InstanceId:            aws.String(request.EC2InstanceID),
			LifecycleHookName:     aws.String(request.LifecycleHookName),
			LifecycleActionResult: aws.String(request.Params["LifecycleActionResult"]),
		},
	)
	return response, err
}

func main() {
	lambda.Start(putLifecycleAction)
}
