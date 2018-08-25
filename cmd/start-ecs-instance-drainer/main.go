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

func startECSInstanceDrainer(request internal.DrainParameters) error {
	var err error

	drainParameters := request

	drainParameters.StateMachineARN = os.Getenv("STATE_MACHINE_ARN")
	if drainParameters.StateMachineARN == "" {
		return errors.New("STATE_MACHINE_ARN environment variable not defined")
	}

	drainParameters.ECSCluster = os.Getenv("ECS_CLUSTER")
	if drainParameters.ECSCluster == "" {
		return errors.New("ECS_CLUSTER environment variable not defined")
	}

	var timeout time.Duration
	if os.Getenv("TIMEOUT") != "" {
		timeout, err = time.ParseDuration(os.Getenv("TIMEOUT"))
		if err != nil {
			return err
		}
	}
	drainParameters.Deadline = time.Now().Add(timeout).Format(time.RFC3339)

	sess := session.Must(session.NewSession())
	drainParameters.ECSInstanceID, err = internal.GetECSInstanceARN(sess, request.ECSCluster, request.AutoScalingGroupName)
	if err != nil {
		return errors.WithMessage(err, "GetECSInstanceARN")
	}
	if drainParameters.ECSInstanceID == "" {
		return errors.Errorf("No ECS instance ID in cluster %s found for EC2 instance ID %s", request.ECSCluster, request.EC2InstanceID)
	}

	fmt.Printf("Setting ECS instance %s on cluster %s to DRAINING state\n", drainParameters.ECSInstanceID, drainParameters.ECSCluster)

	ecsClient := ecs.New(sess)
	if _, err := ecsClient.UpdateContainerInstancesState(
		&ecs.UpdateContainerInstancesStateInput{
			Cluster:            aws.String(drainParameters.ECSCluster),
			ContainerInstances: aws.StringSlice([]string{drainParameters.ECSInstanceID}),
			Status:             aws.String("DRAINING"),
		},
	); err != nil {
		return errors.WithMessage(err, "UpdateContainerInstancesState")
	}

	switch strings.ToLower(os.Getenv("STOP_ALL_NON_SERVICE_TASKS")) {
	case "1", "true", "t", "yes", "y":
		fmt.Printf("Stopping all non-service tasks on ECS instance %s in cluster %s\n", drainParameters.ECSInstanceID, drainParameters.ECSCluster)
		if err := stopAllNonServiceTasks(sess, request.ECSCluster, drainParameters.ECSInstanceID); err != nil {
			return errors.WithMessage(err, "stopAllNonServiceTasks")
		}
	}

	groups := strings.Split(os.Getenv("STOP_TASK_GROUPS"), ",")
	if len(groups) > 0 {
		fmt.Printf("Stopping tasks in groups %s on ECS instance %s in cluster %s\n", os.Getenv("STOP_TASK_GROUPS"), drainParameters.ECSInstanceID, drainParameters.ECSCluster)
		if err := stopTaskGroups(sess, request.ECSCluster, drainParameters.ECSInstanceID, groups); err != nil {
			return errors.WithMessage(err, "stopTaskGroups")
		}
	}

	startTime := time.Now()
	executionName := startTime.Format("20060102T150405Z0700")

	sfnInput, err := json.Marshal(drainParameters)
	if err != nil {
		return errors.WithMessage(err, "Error marshaling JSON")
	}

	sfnClient := sfn.New(sess)
	_, err = sfnClient.StartExecution(&sfn.StartExecutionInput{
		Name:            aws.String(executionName),
		StateMachineArn: aws.String(drainParameters.StateMachineARN),
		Input:           aws.String(string(sfnInput)),
	})
	if err != nil {
		return errors.WithMessage(err, "StartExecution")
	}

	fmt.Printf("Started Step Function %s with execution name %s\n", drainParameters.StateMachineARN, executionName)
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
