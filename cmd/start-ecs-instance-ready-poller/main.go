package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/pkg/errors"
)

func startECSInstancePoller(event internal.CloudwatchLifecycleEvent) error {
	var err error

	params := internal.ECSReadyParameters{}
	params.AutoScalingGroupName = event.Detail.AutoScalingGroupName
	params.EC2InstanceID = event.Detail.EC2InstanceID
	params.LifecycleActionToken = event.Detail.LifecycleActionToken
	params.LifecycleHookName = event.Detail.LifecycleHookName
	params.LifecycleTransition = event.Detail.LifecycleTransition

	params.StateMachineARN = os.Getenv("STATE_MACHINE_ARN")
	if params.StateMachineARN == "" {
		return errors.New("STATE_MACHINE_ARN environment variable not defined")
	}

	params.ECSCluster = os.Getenv("ECS_CLUSTER")
	if params.ECSCluster == "" {
		return errors.New("ECS_CLUSTER environment variable not defined")
	}

	var timeout time.Duration
	if os.Getenv("TIMEOUT") != "" {
		timeout, err = time.ParseDuration(os.Getenv("TIMEOUT"))
		if err != nil {
			return err
		}
	}
	params.Deadline = time.Now().Add(timeout).Format(time.RFC3339)

	sess := session.Must(session.NewSession())

	params.RequiredTaskFamilies = strings.Split(os.Getenv("REQUIRED_TASK_FAMILIES"), ",")

	startTime := time.Now()
	executionName := startTime.Format("20060102T150405Z0700")

	sfnInput, err := json.Marshal(params)
	if err != nil {
		return errors.WithMessage(err, "Error marshaling JSON")
	}

	sfnClient := sfn.New(sess)
	if _, err := sfnClient.StartExecution(&sfn.StartExecutionInput{
		Name:            aws.String(executionName),
		StateMachineArn: aws.String(params.StateMachineARN),
		Input:           aws.String(string(sfnInput)),
	}); err != nil {
		return errors.WithMessage(err, "StartExecution")
	}

	fmt.Printf("Started Step Function %s with execution name %s\n", params.StateMachineARN, executionName)
	fmt.Printf("Input:\n%s\n", sfnInput)
	return nil
}

func main() {
	lambda.Start(startECSInstancePoller)
}
