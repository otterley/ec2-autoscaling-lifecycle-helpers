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

func TestECSInstanceReady(t *testing.T) {
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
	ecsCluster := terraform.Output(t, tfOpts, "ecs_cluster_name")

	t.Run("Cloudwatch Event Connected",
		testCloudwatchEventConnected(ctx, terraform.Output(t, tfOpts, "start_poller_lambda_arn")))

	err := setASGDesiredCapacity(ctx, autoScalingGroupName, 1)
	assert.NoError(t, err)

	t.Run("Instance transitions to Pending:Wait state", testInstanceTransition(ctx, autoScalingGroupName, "Pending:Wait"))

	t.Run("Step Function Started",
		testStepFunctionStarted(ctx, terraform.Output(t, tfOpts, "step_function_arn")))

	t.Run("Instance joined ECS cluster", testInstanceJoinedECSCluster(ctx, ecsCluster, 1))

	t.Run("Instance transitions to InService state", testInstanceTransition(ctx, autoScalingGroupName, "InService"))
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

func testInstanceJoinedECSCluster(ctx context.Context, cluster string, activeMin int) func(t *testing.T) {
	return func(t *testing.T) {
		client := ecs.New(session.Must(session.NewSession()))
		for {
			output, err := client.ListContainerInstancesWithContext(
				ctx,
				&ecs.ListContainerInstancesInput{
					Cluster: aws.String(cluster),
					Status:  aws.String("ACTIVE"),
				},
			)
			assert.NoError(t, err)
			fmt.Printf("Cluster %s ACTIVE instance count: %d\n", cluster, len(output.ContainerInstanceArns))
			if len(output.ContainerInstanceArns) >= activeMin {
				return
			}
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

func testInstanceTransition(ctx context.Context, autoScalingGroupName, state string) func(t *testing.T) {
	return func(t *testing.T) {
		client := autoscaling.New(session.Must(session.NewSession()))

		for {
			count := 0

			groups, err := client.DescribeAutoScalingGroupsWithContext(
				ctx,
				&autoscaling.DescribeAutoScalingGroupsInput{
					AutoScalingGroupNames: aws.StringSlice([]string{autoScalingGroupName}),
				},
			)
			assert.NoError(t, err)

			var instanceIDs []*string
			for _, instanceID := range groups.AutoScalingGroups[0].Instances {
				instanceIDs = append(instanceIDs, instanceID.InstanceId)
			}

			instances, err := client.DescribeAutoScalingInstancesWithContext(
				ctx,
				&autoscaling.DescribeAutoScalingInstancesInput{
					InstanceIds: instanceIDs,
				},
			)
			assert.NoError(t, err)

			for _, instance := range instances.AutoScalingInstances {
				if aws.StringValue(instance.LifecycleState) == state {
					count++
				}
			}

			fmt.Printf("Instances in %s state: %d\n", state, count)
			if count > 0 {
				return
			}

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
