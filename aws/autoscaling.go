package aws

type AutoscalingGroups []*AutoscalingGroupMonitor

type autoscalingGroup struct {
	autoscalingGroupName string
	launchConfiguration  string
	desiredCapacity      int64
	instanceMonitors     map[string]*InstanceMonitor
}

type AutoscalingGroupMonitor struct {
	autoscaling   *autoscalingGroup
	awsConnection AwsConnectionInterface
}

func NewAutoscalingGroups(conn AwsConnectionInterface, autoscalingGroupNameList []string) (*AutoscalingGroups, error) {

	autoscalingGroups := new(AutoscalingGroups)

	for _, autoscalingGroupName := range autoscalingGroupNameList {

		autoscalingGroup, _ := NewAutoscalingGroupMonitor(conn, autoscalingGroupName)
		*autoscalingGroups = append(*autoscalingGroups, autoscalingGroup)
	}

	return autoscalingGroups, nil
}

func NewAutoscalingGroupMonitor(awsConnection AwsConnectionInterface, autoscalingGroupName string) (*AutoscalingGroupMonitor, error) {

	response, err := awsConnection.DescribeAGByName(autoscalingGroupName)

	if err != nil {
		return &AutoscalingGroupMonitor{}, err
	}

	instanceMonitors := make(map[string]*InstanceMonitor)

	for _, instance := range response.Instances {
		instanceMonitor, _ := NewInstanceMonitor(awsConnection, autoscalingGroupName, *instance.LaunchConfigurationName, *instance.InstanceId)
		instanceMonitors[*instance.InstanceId] = instanceMonitor
	}

	return &AutoscalingGroupMonitor{
		autoscaling: &autoscalingGroup{
			autoscalingGroupName: autoscalingGroupName,
			launchConfiguration:  *response.LaunchConfigurationName,
			desiredCapacity:      *response.DesiredCapacity,
			instanceMonitors:     instanceMonitors,
		},
		awsConnection: awsConnection,
	}, nil
}

func (a *AutoscalingGroupMonitor) Refresh() error {

	response, err := a.awsConnection.DescribeAGByName(a.autoscaling.autoscalingGroupName)

	if err != nil {
		return err
	}

	for _, instance := range response.Instances {
		_, ok := a.autoscaling.instanceMonitors[*instance.InstanceId]
		if !ok {
			instanceMonitor, _ := NewInstanceMonitor(a.awsConnection, a.autoscaling.autoscalingGroupName, *instance.LaunchConfigurationName, *instance.InstanceId)
			a.autoscaling.instanceMonitors[*instance.InstanceId] = instanceMonitor
		}
	}

	var found bool
	for instanceId, _ := range a.autoscaling.instanceMonitors {
		found = false
		for _, instance := range response.Instances {
			if *instance.InstanceId == instanceId {
				found = true
				break
			}
		}
		if !found {
			delete(a.autoscaling.instanceMonitors, instanceId)
		}

	}

	return nil
}

func (a *AutoscalingGroupMonitor) GetInstances() *[]InstanceMonitor {

	instanceMonitor := []InstanceMonitor{}

	for _, value := range a.autoscaling.instanceMonitors {
		instanceMonitor = append(instanceMonitor, *value)
	}

	return &instanceMonitor
}

func (a *AutoscalingGroupMonitor) NumUndesiredInstances() int {

	if len(a.autoscaling.instanceMonitors) > int(a.autoscaling.desiredCapacity) {
		return len(a.autoscaling.instanceMonitors) - int(a.autoscaling.desiredCapacity)
	}

	return 0
}

func (a *AutoscalingGroupMonitor) hasInstance(instanceId string) bool {

	for _, instanceMonitor := range a.autoscaling.instanceMonitors {
		if instanceMonitor.instance.instanceId == instanceId {
			return true
		}
	}

	return false
}
