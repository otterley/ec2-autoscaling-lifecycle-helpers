package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
)

func countECSTasks(request internal.DrainParameters) (internal.DrainParameters, error) {
	response := request
	response.ECSTaskCount = 0

	client := ecs.New(session.Must(session.NewSession()))
	return response, client.ListTasksPages(
		&ecs.ListTasksInput{
			Cluster:           aws.String(request.ECSCluster),
			ContainerInstance: aws.String(request.ECSInstanceID),
			DesiredStatus:     aws.String("RUNNING"),
		},
		func(page *ecs.ListTasksOutput, lastPage bool) bool {
			response.ECSTaskCount += len(page.TaskArns)
			return !lastPage
		},
	)
}

func main() {
	lambda.Start(countECSTasks)
}
