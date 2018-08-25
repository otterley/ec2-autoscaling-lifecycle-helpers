package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/pkg/errors"
)

func countRunningExecutions(request internal.DrainParameters) (response internal.DrainParameters, err error) {
	response = request
	response.RunningExecutionCount = 0

	sess := session.Must(session.NewSession())

	client := sfn.New(sess)
	if err := client.ListExecutionsPages(
		&sfn.ListExecutionsInput{
			StateMachineArn: aws.String(request.StateMachineARN),
			StatusFilter:    aws.String("RUNNING"),
		},
		func(result *sfn.ListExecutionsOutput, lastPage bool) bool {
			response.RunningExecutionCount += len(result.Executions)
			return !lastPage
		},
	); err != nil {
		return response, errors.WithMessage(err, "ListExecutions")
	}

	return
}

func main() {
	lambda.Start(countRunningExecutions)
}
