package main

import (
	"time"

	"github.com/pkg/errors"

	"github.com/aws/aws-lambda-go/lambda"
)

func checkDeadline(request map[string]interface{}) (map[string]interface{}, error) {
	response := request
	deadline, err := time.Parse(time.RFC3339, request["Deadline"].(string))
	if err != nil {
		return response, errors.WithMessage(err, "time.Parse")
	}
	if time.Now().After(deadline) {
		response["PastDeadline"] = true
	}
	return response, nil
}

func main() {
	lambda.Start(checkDeadline)
}
