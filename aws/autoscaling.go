package aws

import "github.com/aws/aws-sdk-go/service/autoscaling"

type AutoscalingGroups struct {
	monitors	map[string]map[string]*AutoscalingGroupMonitor
	awsConnection 	AwsConnectionInterface
}

type AutoscalingGroupMonitor struct {
	autoscaling   *autoscalingGroup
	awsConnection AwsConnectionInterface
}

type autoscalingGroup struct {
	autoscalingGroupName string
	launchConfiguration  string
	desiredCapacity      int64
	instanceMonitors     map[string]*InstanceMonitor
}

func NewAutoscalingGroups(awsConnection AwsConnectionInterface, autoscalingGroupNameList []string) (*AutoscalingGroups, error) {

	monitors := map[string]map[string]*AutoscalingGroupMonitor{}
	for _, autoscalingGroupName := range autoscalingGroupNameList {
		monitors[autoscalingGroupName] = map[string]*AutoscalingGroupMonitor{}
	}

	autoscalingGroups := &AutoscalingGroups{
		monitors: monitors,
		awsConnection: awsConnection,
	}

	return autoscalingGroups, nil
}

func NewAutoscalingGroupMonitor(awsConnection AwsConnectionInterface, autoscalingGroupName string) (*AutoscalingGroupMonitor, error) {

	return &AutoscalingGroupMonitor{
		autoscaling: &autoscalingGroup{
			autoscalingGroupName: autoscalingGroupName,
			launchConfiguration:  "",
			desiredCapacity:      0,
			instanceMonitors:     map[string]*InstanceMonitor{},
		},
		awsConnection: awsConnection,
	}, nil
}

func (a *AutoscalingGroups) Refresh() error {

	for autoscalingGroupPrefix, _ := range a.monitors {

		response, err := a.awsConnection.DescribeAGByName(autoscalingGroupPrefix)
		if err != nil {
			return err
		}

		for _, autoscalingGroupResponse := range response {
			_, ok := a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName]
			if ok {
				a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName].Refresh(autoscalingGroupResponse)
			} else {
				autoscalingGroupMonitor, _ := NewAutoscalingGroupMonitor(a.awsConnection, *autoscalingGroupResponse.AutoScalingGroupName)
				autoscalingGroupMonitor.Refresh(autoscalingGroupResponse)
				a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName] = autoscalingGroupMonitor
			}
		}

		var found bool
		for autoscalingGroupName, _ := range a.monitors[autoscalingGroupPrefix] {
			found = false
			for _, autoscalingGroupResponse := range response {
				if autoscalingGroupName == *autoscalingGroupResponse.AutoScalingGroupName {
					found = true
					break
				}
			}
			if !found {
				delete(a.monitors[autoscalingGroupPrefix], autoscalingGroupName)
			}
		}
	}

	return nil
}

func (a *AutoscalingGroups) GetMonitors() []*AutoscalingGroupMonitor {

	var monitors = []*AutoscalingGroupMonitor{}

	for autoscalingGroupPrefix := range a.monitors {
		for autoscalingGroupName := range a.monitors[autoscalingGroupPrefix] {
			monitors = append(monitors, a.monitors[autoscalingGroupPrefix][autoscalingGroupName])
		}
	}

	return monitors
}

func (a *AutoscalingGroupMonitor) Refresh(autoscalingGroup *autoscaling.Group) error {

	if !*autoscalingGroup.NewInstancesProtectedFromScaleIn {
		instancesToProtect := []*string{}

		for _, instance := range autoscalingGroup.Instances {
				instancesToProtect = append(instancesToProtect, instance.InstanceId)
		}

		err := a.awsConnection.SetASGInstanceProtection(autoscalingGroup.AutoScalingGroupName, instancesToProtect)
		if err != nil {
			return err
		}
	}

	a.autoscaling.launchConfiguration = *autoscalingGroup.LaunchConfigurationName
	a.autoscaling.desiredCapacity = *autoscalingGroup.DesiredCapacity

	for _, instance := range autoscalingGroup.Instances {
		_, ok := a.autoscaling.instanceMonitors[*instance.InstanceId]
		if !ok {
			instanceMonitor, _ := NewInstanceMonitor(a.awsConnection, a.autoscaling.autoscalingGroupName, *instance.LaunchConfigurationName, *instance.InstanceId)
			a.autoscaling.instanceMonitors[*instance.InstanceId] = instanceMonitor
		}
	}

	var found bool
	for instanceId, _ := range a.autoscaling.instanceMonitors {
		found = false
		for _, instance := range autoscalingGroup.Instances {
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

func (a *AutoscalingGroupMonitor) RemoveInstance(instanceMonitor *InstanceMonitor) {

	delete(a.autoscaling.instanceMonitors, instanceMonitor.instance.instanceId)
}
