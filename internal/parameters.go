package internal

type StartDrainEvent struct {
	Detail DrainParameters `json:"detail"`
}

type DrainParameters struct {
	// These come directly from the CloudWatch Event -- see
	// https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/EventTypes.html#auto_scaling_event_types
	LifecycleActionToken string
	AutoScalingGroupName string
	LifecycleHookName    string
	EC2InstanceID        string `json:"EC2InstanceId"`
	LifecycleTransition  string

	// Added by start function
	StateMachineARN string
	Deadline        string
	PastDeadline    bool
	ECSInstanceID   string
	ECSCluster      string
	ECSTaskCount    int

	// Added by counter
	RunningExecutionCount int

	// Added by Step Function
	Params map[string]string
}

type StartECSReadyPollEvent struct {
	Detail ECSReadyParameters `json:"detail"`
}

type ECSReadyParameters struct {
	// These come directly from the CloudWatch Event -- see
	// https://docs.aws.amazon.com/AmazonCloudWatch/latest/events/EventTypes.html#auto_scaling_event_types
	LifecycleActionToken string
	AutoScalingGroupName string
	LifecycleHookName    string
	EC2InstanceID        string `json:"EC2InstanceId"`
	LifecycleTransition  string

	// Added by start function
	StateMachineARN      string
	Deadline             string
	PastDeadline         bool
	ECSCluster           string
	ECSInstanceID        string
	RequiredTaskFamilies []string
	Ready                bool

	// Added by counter
	RunningExecutionCount int

	// Added by Step Function
	Params map[string]string
}
