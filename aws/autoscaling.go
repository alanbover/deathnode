package aws

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
	log "github.com/sirupsen/logrus"
)

// monitors is a map of [ASGprefix][ASGname]AutoscalingGroupMonitor
type AutoscalingGroups struct {
	monitors      map[string]map[string]*AutoscalingGroupMonitor
	awsConnection AwsConnectionInterface
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
		monitors:      monitors,
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

		if len(response) == 0 {
			log.Warnf("No autoscaling groups found under autoscalingGroupPrefix %s", autoscalingGroupPrefix)
		}

		for _, autoscalingGroupResponse := range response {
			_, ok := a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName]
			if ok {
				a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName].Refresh(autoscalingGroupResponse)
			} else {
				log.Infof("Found new autoscalingGroup to monitor: %s", *autoscalingGroupResponse.AutoScalingGroupName)
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
				log.Infof("Autoscaling group %s removed. Deleting it", autoscalingGroupName)
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

func (a *AutoscalingGroups) GetAutoscalingNameByInstanceId(instanceId string) (string, bool) {

	for asgPrefix, _ := range a.monitors {
		for _, asgGroupMonitor := range a.monitors[asgPrefix] {
			instances := asgGroupMonitor.GetInstancesNotMarkedToBeRemoved()
			for _, instanceMonitor := range instances {
				if instanceMonitor.instance.instanceId == instanceId {
					return asgGroupMonitor.autoscaling.autoscalingGroupName, true
				}
			}
		}
	}

	return "", false

}

func (a *AutoscalingGroupMonitor) Refresh(autoscalingGroup *autoscaling.Group) error {

	if !*autoscalingGroup.NewInstancesProtectedFromScaleIn {
		log.Infof("Setting autoscaling %s and it's instances scaleInProtection flag", *autoscalingGroup.AutoScalingGroupName)
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
			log.Debugf("Found new instance to monitor in autoscaling %s: %s", a.autoscaling.autoscalingGroupName, *instance.InstanceId)
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
			log.Debugf("Instance %s has dissapeared from ASG %s. Stop monitoring it", instanceId, a.autoscaling.autoscalingGroupName)
			delete(a.autoscaling.instanceMonitors, instanceId)
		}

	}

	return nil
}

func (a *AutoscalingGroupMonitor) NumUndesiredInstances() int {

	if len(a.autoscaling.instanceMonitors)-len(a.GetInstancesMarkedToBeRemoved()) > int(a.autoscaling.desiredCapacity) {
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

func (a *AutoscalingGroupMonitor) GetInstancesMarkedToBeRemoved() []*InstanceMonitor {

	instancesMarkedToBeRemoved := []*InstanceMonitor{}
	for _, instanceMonitor := range a.autoscaling.instanceMonitors {
		if instanceMonitor.instance.markedToBeRemoved {
			instancesMarkedToBeRemoved = append(instancesMarkedToBeRemoved, instanceMonitor)
		}
	}

	return instancesMarkedToBeRemoved
}

func (a *AutoscalingGroupMonitor) GetInstancesNotMarkedToBeRemoved() []InstanceMonitor {

	instancesNotMarkedToBeRemoved := []InstanceMonitor{}
	for _, instanceMonitor := range a.autoscaling.instanceMonitors {
		if !instanceMonitor.instance.markedToBeRemoved {
			instancesNotMarkedToBeRemoved = append(instancesNotMarkedToBeRemoved, *instanceMonitor)
		}
	}

	return instancesNotMarkedToBeRemoved
}