package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
)

func recordLifecycleHeartbeat(request internal.DrainParameters) (internal.DrainParameters, error) {
	response := request
	client := autoscaling.New(session.Must(session.NewSession()))

	_, err := client.RecordLifecycleActionHeartbeat(
		&autoscaling.RecordLifecycleActionHeartbeatInput{
			AutoScalingGroupName: aws.String(request.AutoScalingGroupName),
			InstanceId:           aws.String(request.EC2InstanceID),
			LifecycleHookName:    aws.String(request.LifecycleHookName),
		},
	)
	return response, err
}

func main() {
	lambda.Start(recordLifecycleHeartbeat)
}
