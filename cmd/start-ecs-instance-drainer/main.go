package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/pkg/errors"
)

func startECSInstanceDrainer(event internal.CloudwatchLifecycleEvent) error {
	var err error

	params := internal.DrainParameters{}
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
	params.ECSInstanceID, err = internal.GetECSInstanceARN(sess, params.ECSCluster, params.EC2InstanceID)
	if err != nil {
		return errors.WithMessage(err, "GetECSInstanceARN")
	}
	if params.ECSInstanceID == "" {
		return fmt.Errorf("No ECS instance matching EC2 instance ID %s found in cluster %s", params.EC2InstanceID, params.ECSCluster)
	}

	fmt.Printf("Setting ECS instance %s on cluster %s to DRAINING state\n", params.ECSInstanceID, params.ECSCluster)

	ecsClient := ecs.New(sess)
	if _, err := ecsClient.UpdateContainerInstancesState(
		&ecs.UpdateContainerInstancesStateInput{
			Cluster:            aws.String(params.ECSCluster),
			ContainerInstances: aws.StringSlice([]string{params.ECSInstanceID}),
			Status:             aws.String("DRAINING"),
		},
	); err != nil {
		return errors.WithMessage(err, "UpdateContainerInstancesState")
	}

	switch strings.ToLower(os.Getenv("STOP_ALL_NON_SERVICE_TASKS")) {
	case "1", "true", "t", "yes", "y":
		fmt.Printf("Stopping all non-service tasks on ECS instance %s in cluster %s\n", params.ECSInstanceID, params.ECSCluster)
		if err := stopAllNonServiceTasks(sess, params.ECSCluster, params.ECSInstanceID); err != nil {
			return errors.WithMessage(err, "stopAllNonServiceTasks")
		}
	}

	groups := strings.Split(os.Getenv("STOP_TASK_GROUPS"), ",")
	if len(groups) > 0 {
		fmt.Printf("Stopping tasks in groups %s on ECS instance %s in cluster %s\n", os.Getenv("STOP_TASK_GROUPS"), params.ECSInstanceID, params.ECSCluster)
		if err := stopTaskGroups(sess, params.ECSCluster, params.ECSInstanceID, groups); err != nil {
			return errors.WithMessage(err, "stopTaskGroups")
		}
	}

	startTime := time.Now()
	executionName := startTime.Format("20060102T150405Z0700")

	sfnInput, err := json.Marshal(params)
	if err != nil {
		return errors.WithMessage(err, "Error marshaling JSON")
	}

	sfnClient := sfn.New(sess)
	_, err = sfnClient.StartExecution(&sfn.StartExecutionInput{
		Name:            aws.String(executionName),
		StateMachineArn: aws.String(params.StateMachineARN),
		Input:           aws.String(string(sfnInput)),
	})
	if err != nil {
		return errors.WithMessage(err, "StartExecution")
	}

	fmt.Printf("Started Step Function %s with execution name %s\n", params.StateMachineARN, executionName)
	fmt.Printf("Input:\n%s\n", sfnInput)
	return nil
}

func stopAllNonServiceTasks(sess client.ConfigProvider, cluster, ecsInstanceID string) error {
	return stopMatchingTasks(sess, cluster, ecsInstanceID,
		func(task *ecs.Task) bool {
			return !strings.HasPrefix(aws.StringValue(task.Group), "service:")
		})
}

func stopTaskGroups(sess client.ConfigProvider, cluster, ecsInstanceID string, taskGroups []string) error {
	return stopMatchingTasks(sess, cluster, ecsInstanceID,
		func(task *ecs.Task) bool {
			for _, group := range taskGroups {
				if aws.StringValue(task.Group) == group {
					return true
				}
			}
			return false
		})
}

func stopMatchingTasks(sess client.ConfigProvider, cluster, ecsInstanceID string, matchFunc func(*ecs.Task) bool) error {
	var innerErr error
	client := ecs.New(sess)
	if err := client.ListTasksPages(
		&ecs.ListTasksInput{
			Cluster:           aws.String(cluster),
			ContainerInstance: aws.String(ecsInstanceID),
		},
		func(page *ecs.ListTasksOutput, lastPage bool) bool {
			tasks, innerErr := client.DescribeTasks(
				&ecs.DescribeTasksInput{
					Cluster: aws.String(cluster),
					Tasks:   page.TaskArns,
				},
			)
			if innerErr != nil {
				return false // abandon ship
			}
			for _, task := range tasks.Tasks {
				if matchFunc(task) {
					_, innerErr = client.StopTask(
						&ecs.StopTaskInput{
							Cluster: aws.String(cluster),
							Task:    task.TaskArn,
							Reason:  aws.String("ECS instance drainer requested stop"),
						},
					)
					if innerErr != nil {
						return false
					}
				}
			}
			return !lastPage
		},
	); err != nil {
		return err
	}
	return innerErr
}

func main() {
	lambda.Start(startECSInstanceDrainer)
}
