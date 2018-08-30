package internal

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pkg/errors"
)

var ecsInstanceARNCache map[string]string

func GetECSInstanceARN(sess client.ConfigProvider, cluster, ec2InstanceID string) (string, error) {
	var (
		arn      string
		innerErr error
	)

	// Return a cached response if possible
	if arn, ok := ecsInstanceARNCache[ec2InstanceID]; ok {
		return arn, nil
	}

	client := ecs.New(sess)
	if err := client.ListContainerInstancesPages(
		&ecs.ListContainerInstancesInput{
			Cluster: aws.String(cluster),
		},
		func(page *ecs.ListContainerInstancesOutput, lastPage bool) bool {
			if len(page.ContainerInstanceArns) == 0 {
				return false // nothing to do
			}
			var instances *ecs.DescribeContainerInstancesOutput
			instances, innerErr = client.DescribeContainerInstances(
				&ecs.DescribeContainerInstancesInput{
					Cluster:            aws.String(cluster),
					ContainerInstances: page.ContainerInstanceArns,
				},
			)
			if innerErr != nil {
				return false
			}
			for _, instance := range instances.ContainerInstances {
				if aws.StringValue(instance.Ec2InstanceId) == ec2InstanceID {
					arn = aws.StringValue(instance.ContainerInstanceArn)
					return false // we're done
				}
			}
			return !lastPage
		},
	); err != nil {
		return "", errors.WithMessage(err, "ListContainerInstances")
	}
	if arn != "" {
		// Write to cache
		ecsInstanceARNCache[ec2InstanceID] = arn
	}
	return arn, errors.WithMessage(innerErr, "DescribeContainerInstances")
}
