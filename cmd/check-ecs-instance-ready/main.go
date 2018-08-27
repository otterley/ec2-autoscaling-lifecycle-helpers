package main

import (
	"fmt"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/pkg/errors"
)

func checkECSInstanceReady(request internal.ECSReadyParameters) (internal.ECSReadyParameters, error) {
	response := request
	response.Ready = false

	sess := session.Must(session.NewSession())
	client := ecs.New(sess)

	if request.ECSInstanceID == "" {
		var err error
		request.ECSInstanceID, err = internal.GetECSInstanceARN(sess, request.ECSCluster, request.EC2InstanceID)
		if err != nil {
			return response, errors.WithMessage(err, "GetECSInstanceARN")
		}
		if request.ECSInstanceID == "" {
			// No instance ID assigned yet, so not ready
			fmt.Printf("No ECS instance ID in cluster %s for EC2 instance ID %s\n", request.ECSCluster, request.EC2InstanceID)
			return response, nil
		}
	}

	result, err := client.DescribeContainerInstances(
		&ecs.DescribeContainerInstancesInput{
			Cluster:            aws.String(request.ECSCluster),
			ContainerInstances: aws.StringSlice([]string{request.ECSInstanceID}),
		},
	)
	if err != nil {
		return response, errors.WithMessage(err, "DescribeContainerInstances")
	}
	if len(result.ContainerInstances) != 1 {
		return response, errors.New("assertion failure: container instances != 1")
	}

	if !aws.BoolValue(result.ContainerInstances[0].AgentConnected) ||
		aws.StringValue(result.ContainerInstances[0].Status) != "ACTIVE" {
		fmt.Printf("ECS instance %s not connected or state not ACTIVE\n", request.ECSInstanceID)
		return response, nil
	}

	for _, family := range request.RequiredTaskFamilies {
		taskCount := 0
		if err := client.ListTasksPages(
			&ecs.ListTasksInput{
				Cluster:           aws.String(request.ECSCluster),
				ContainerInstance: aws.String(request.ECSInstanceID),
				DesiredStatus:     aws.String("RUNNING"),
				Family:            aws.String(family),
			},
			func(page *ecs.ListTasksOutput, lastPage bool) bool {
				taskCount += len(page.TaskArns)
				return !lastPage
			},
		); err != nil {
			return response, errors.WithMessage(err, "ListTasks")
		}
		fmt.Printf("Task count for family %s on ECS instance %s: %d\n", family, request.ECSInstanceID, taskCount)
		if taskCount == 0 {
			fmt.Println("ECS instance not ready")
			return response, nil
		}
	}

	fmt.Println("ECS instance ready")
	response.Ready = true
	return response, nil
}

func main() {
	lambda.Start(checkECSInstanceReady)
}
