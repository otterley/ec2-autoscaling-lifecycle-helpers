package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func putLifecycleAction(request map[string]interface{}) (map[string]interface{}, error) {
	response := request
	client := autoscaling.New(session.Must(session.NewSession()))

	params := request["Params"].(map[string]interface{})

	_, err := client.CompleteLifecycleAction(
		&autoscaling.CompleteLifecycleActionInput{
			AutoScalingGroupName:  aws.String(request["AutoScalingGroupName"].(string)),
			InstanceId:            aws.String(request["EC2InstanceId"].(string)),
			LifecycleHookName:     aws.String(request["LifecycleHookName"].(string)),
			LifecycleActionResult: aws.String(params["LifecycleActionResult"].(string)),
		},
	)
	return response, err
}

func main() {
	lambda.Start(putLifecycleAction)
}
