package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/pkg/errors"
)

func countRunningExecutions(request map[string]interface{}) (response map[string]interface{}, err error) {
	response = request
	count := 0

	sess := session.Must(session.NewSession())

	client := sfn.New(sess)
	if err := client.ListExecutionsPages(
		&sfn.ListExecutionsInput{
			StateMachineArn: aws.String(request["StateMachineARN"].(string)),
			StatusFilter:    aws.String("RUNNING"),
		},
		func(result *sfn.ListExecutionsOutput, lastPage bool) bool {
			count += len(result.Executions)
			return !lastPage
		},
	); err != nil {
		return response, errors.WithMessage(err, "ListExecutions")
	}

	response["RunningExecutionCount"] = count

	return
}

func main() {
	lambda.Start(countRunningExecutions)
}
