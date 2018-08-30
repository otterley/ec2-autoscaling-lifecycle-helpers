package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func recordLifecycleHeartbeat(request map[string]interface{}) (map[string]interface{}, error) {
	response := request
	client := autoscaling.New(session.Must(session.NewSession()))

	_, err := client.RecordLifecycleActionHeartbeat(
		&autoscaling.RecordLifecycleActionHeartbeatInput{
			AutoScalingGroupName: aws.String(request["AutoScalingGroupName"].(string)),
			InstanceId:           aws.String(request["EC2InstanceId"].(string)),
			LifecycleHookName:    aws.String(request["LifecycleHookName"].(string)),
		},
	)
	return response, err
}

func main() {
	lambda.Start(recordLifecycleHeartbeat)
}
