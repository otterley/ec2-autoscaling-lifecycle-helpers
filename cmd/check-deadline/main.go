package main

import (
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
)

func checkDeadline(request internal.DrainParameters) (internal.DrainParameters, error) {
	response := request
	deadline, err := time.Parse(time.RFC3339, request.Deadline)
	if err != nil {
		return response, errors.WithMessage(err, "time.Parse")
	}
	if time.Now().After(deadline) {
		response.PastDeadline = true
	}
	return response, nil
}

func main() {
	lambda.Start(checkDeadline)
}
