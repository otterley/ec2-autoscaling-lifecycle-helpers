package ecs_instance_drainer_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sfn"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/otterley/ec2-autoscaling-lifecycle-helpers/internal"
	"github.com/stretchr/testify/assert"
)

func TestECSInstanceDrainer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(20*time.Minute))
	defer cancel()

	tfOpts := &terraform.Options{
		Vars: map[string]interface{}{
			"lambda_version": internal.MustEnv("LAMBDA_VERSION"),
		},
	}
	defer terraform.Destroy(t, tfOpts)
	terraform.InitAndApply(t, tfOpts)

	autoScalingGroupName := terraform.Output(t, tfOpts, "autoscaling_group_name")

	t.Run("Cloudwatch Event Connected",
		testCloudwatchEventConnected(ctx, terraform.Output(t, tfOpts, "start_drainer_lambda_arn")))

	if err := setASGDesiredCapacity(ctx, autoScalingGroupName, 1); err != nil {
		t.Fatal(err)
	}

	t.Run("Step Function Started",
		testStepFunctionStarted(ctx, terraform.Output(t, tfOpts, "step_function_arn")))

	t.Run("ECS Instance Starts Draining",
		testECSInstanceDraining(ctx, terraform.Output(t, tfOpts, "ecs_cluster_name")))
}

func setASGDesiredCapacity(ctx context.Context, autoScalingGroupName string, desiredCapacity int64) error {
	client := autoscaling.New(session.Must(session.NewSession()))
	_, err := client.UpdateAutoScalingGroupWithContext(
		ctx,
		&autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(autoScalingGroupName),
			DesiredCapacity:      aws.Int64(desiredCapacity),
		},
	)
	return err
}

func testCloudwatchEventConnected(ctx context.Context, targetARN string) func(t *testing.T) {
	return func(t *testing.T) {
		client := cloudwatchevents.New(session.Must(session.NewSession()))
		output, err := client.ListRuleNamesByTargetWithContext(
			ctx,
			&cloudwatchevents.ListRuleNamesByTargetInput{
				TargetArn: aws.String(targetARN),
			},
		)
		assert.NoError(t, err)
		assert.NotEmpty(t, output.RuleNames)
	}
}

func testStepFunctionStarted(ctx context.Context, stateMachineARN string) func(t *testing.T) {
	return func(t *testing.T) {
		client := sfn.New(session.Must(session.NewSession()))

		for {
			executions, err := client.ListExecutionsWithContext(
				ctx,
				&sfn.ListExecutionsInput{
					StateMachineArn: aws.String(stateMachineARN),
				},
			)
			assert.NoError(t, err)

			for _, execution := range executions.Executions {
				if aws.StringValue(execution.Status) == "RUNNING" {
					return
				}
				assert.NotContains(t, aws.StringValue(execution.Status), []string{"FAILED", "TIMED_OUT", "ABORTED"})
			}

			fmt.Printf("Waiting for Step Function %s to start\n", stateMachineARN)
			select {
			case <-ctx.Done():
				// timed out
				return
			case <-time.After(10 * time.Second):
				// check again
			}
		}
	}
}

func testECSInstanceDraining(ctx context.Context, cluster string) func(t *testing.T) {
	return func(t *testing.T) {
		client := ecs.New(session.Must(session.NewSession()))

		for {
			var innerErr error
			drainingCount := 0

			err := client.ListContainerInstancesPagesWithContext(
				ctx,
				&ecs.ListContainerInstancesInput{
					Cluster: aws.String(cluster),
				},
				func(page *ecs.ListContainerInstancesOutput, lastpage bool) bool {
					var instances *ecs.DescribeContainerInstancesOutput
					instances, innerErr = client.DescribeContainerInstancesWithContext(
						ctx,
						&ecs.DescribeContainerInstancesInput{
							Cluster:            aws.String(cluster),
							ContainerInstances: page.ContainerInstanceArns,
						},
					)
					if innerErr != nil {
						return false
					}
					for _, instance := range instances.ContainerInstances {
						if aws.StringValue(instance.Status) == "DRAINING" {
							drainingCount++
						}
					}
					return !lastpage
				},
			)
			assert.NoError(t, err)
			assert.NoError(t, innerErr)

			if drainingCount > 0 {
				return
			}

			fmt.Printf("Waiting for an ECS instance in cluster %s to be set to DRAINING state\n", cluster)
			select {
			case <-ctx.Done():
				// timed out
				return
			case <-time.After(10 * time.Second):
				// check again
			}
		}
	}
}
