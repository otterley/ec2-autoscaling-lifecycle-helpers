package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/pkg/errors"
)

func startKafkaPoller(event internal.CloudwatchLifecycleEvent) error {
	var err error

	params := internal.KafkaReadyParameters{}
	params.AutoScalingGroupName = event.Detail.AutoScalingGroupName
	params.EC2InstanceID = event.Detail.EC2InstanceID
	params.LifecycleActionToken = event.Detail.LifecycleActionToken
	params.LifecycleHookName = event.Detail.LifecycleHookName
	params.LifecycleTransition = event.Detail.LifecycleTransition

	portStr := os.Getenv("KAFKA_PORT")
	if portStr == "" {
		params.KafkaPort = 9092
	} else {
		params.KafkaPort, err = strconv.Atoi(os.Getenv(portStr))
		if err != nil {
			return fmt.Errorf("Failed to parse KAFKA_PORT: %v", err)
		}
	}
	if params.KafkaPort < 0 || params.KafkaPort > 65535 {
		return fmt.Errorf("Kafka port must between 0 and 65535")
	}

	params.StateMachineARN = os.Getenv("STATE_MACHINE_ARN")
	if params.StateMachineARN == "" {
		return errors.New("STATE_MACHINE_ARN environment variable not defined")
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

	params.InternalIPAddr, err = getInternalAddr(sess, params.EC2InstanceID)
	if err != nil {
		return errors.WithMessage(err, "getInternalAddr")
	}

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

func getInternalAddr(sess client.ConfigProvider, ec2InstanceID string) (string, error) {
	ec2Client := ec2.New(sess)
	result, err := ec2Client.DescribeInstances(
		&ec2.DescribeInstancesInput{
			InstanceIds: aws.StringSlice([]string{ec2InstanceID}),
		},
	)
	if err != nil {
		return "", errors.WithMessage(err, "DescribeInstances")
	}
	if len(result.Reservations) != 1 {
		return "", errors.New("assertion failure: reservation count != 1")
	}
	if len(result.Reservations[0].Instances) != 1 {
		return "", errors.New("assertion failure: instance count != 1")
	}
	return aws.StringValue(result.Reservations[0].Instances[0].PrivateIpAddress), nil
}

func main() {
	lambda.Start(startKafkaPoller)
}
