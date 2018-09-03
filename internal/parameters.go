package internal

type CloudwatchLifecycleEvent struct {
	Detail AutoScalingLifecycleEvent `json:"detail"`
}

type AutoScalingLifecycleEvent struct {
	// These come directly from the CloudWatch Event -- see
	// https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/EventTypes.html#auto_scaling_event_types
	LifecycleActionToken string
	AutoScalingGroupName string
	LifecycleHookName    string
	EC2InstanceID        string `json:"EC2InstanceId"`
	LifecycleTransition  string
}

type BaseParameters struct {
	StateMachineARN       string
	Deadline              string
	PastDeadline          bool
	ECSCluster            string
	ECSInstanceID         string
	RunningExecutionCount int
	Params                map[string]string
}

type DrainParameters struct {
	AutoScalingLifecycleEvent
	BaseParameters
	ECSTaskCount int
}

type ECSReadyParameters struct {
	AutoScalingLifecycleEvent
	BaseParameters
	RequiredTaskFamilies []string
	Ready                bool
}
