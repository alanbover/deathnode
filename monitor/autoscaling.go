package monitor

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	log "github.com/sirupsen/logrus"
)

// AutoscalingGroupsMonitor holds a map of [ASGprefix][ASGname]AutoscalingGroupMonitor
type AutoscalingGroupsMonitor struct {
	monitors      map[string]map[string]*AutoscalingGroupMonitor
	awsConnection aws.ClientInterface
	deathNodeMark string
}

// AutoscalingGroupMonitor monitors an AWS autoscaling group, caching it's data
type AutoscalingGroupMonitor struct {
	autoscaling   *autoscalingGroup
	awsConnection aws.ClientInterface
	deathNodeMark string
}

type autoscalingGroup struct {
	autoscalingGroupName string
	desiredCapacity      int64
	instanceMonitors     map[string]*InstanceMonitor
}

// NewAutoscalingGroupMonitors returns an AutoscalingGroups object
func NewAutoscalingGroupMonitors(awsConnection aws.ClientInterface, autoscalingGroupNameList []string, deathNodeMark string) (*AutoscalingGroupsMonitor, error) {

	monitors := map[string]map[string]*AutoscalingGroupMonitor{}
	for _, autoscalingGroupName := range autoscalingGroupNameList {
		monitors[autoscalingGroupName] = map[string]*AutoscalingGroupMonitor{}
	}

	autoscalingGroups := &AutoscalingGroupsMonitor{
		monitors:      monitors,
		awsConnection: awsConnection,
		deathNodeMark: deathNodeMark,
	}

	return autoscalingGroups, nil
}

// NewAutoscalingGroupMonitor returns a "empty" AutoscalingGroupMonitor object
func newAutoscalingGroupMonitor(awsConnection aws.ClientInterface, autoscalingGroupName, deathNodeMark string) (*AutoscalingGroupMonitor, error) {

	return &AutoscalingGroupMonitor{
		autoscaling: &autoscalingGroup{
			autoscalingGroupName: autoscalingGroupName,
			desiredCapacity:      0,
			instanceMonitors:     map[string]*InstanceMonitor{},
		},
		awsConnection: awsConnection,
		deathNodeMark: deathNodeMark,
	}, nil
}

// Refresh updates autoscalingGroups caching all AWS autoscaling groups given the N prefixes
// provided when AutoscalingGroups was created
func (a *AutoscalingGroupsMonitor) Refresh() error {

	for autoscalingGroupPrefix := range a.monitors {

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
				a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName].refresh(autoscalingGroupResponse)
			} else {
				log.Infof("Found new autoscalingGroup to monitor: %s", *autoscalingGroupResponse.AutoScalingGroupName)
				autoscalingGroupMonitor, _ := newAutoscalingGroupMonitor(a.awsConnection, *autoscalingGroupResponse.AutoScalingGroupName, a.deathNodeMark)
				autoscalingGroupMonitor.refresh(autoscalingGroupResponse)
				a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName] = autoscalingGroupMonitor
			}
		}

		var found bool
		for autoscalingGroupName := range a.monitors[autoscalingGroupPrefix] {
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

// GetAllMonitors returns all AutoscalingGroupMonitors cached in AutoscalingGroups
func (a *AutoscalingGroupsMonitor) GetAllMonitors() []*AutoscalingGroupMonitor {

	var monitors = []*AutoscalingGroupMonitor{}

	for autoscalingGroupPrefix := range a.monitors {
		for autoscalingGroupName := range a.monitors[autoscalingGroupPrefix] {
			monitors = append(monitors, a.monitors[autoscalingGroupPrefix][autoscalingGroupName])
		}
	}

	return monitors
}

// GetAutoscalingNameByInstanceID returns the AutoscalingGroupName of the AutoscalingGroup that holds a certain instanceId
func (a *AutoscalingGroupsMonitor) GetAutoscalingNameByInstanceID(instanceID string) (string, bool) {

	for asgPrefix := range a.monitors {
		for _, asgGroupMonitor := range a.monitors[asgPrefix] {
			for _, instanceMonitor := range asgGroupMonitor.autoscaling.instanceMonitors {
				if instanceMonitor.instance.instanceID == instanceID {
					return asgGroupMonitor.autoscaling.autoscalingGroupName, true
				}
			}
		}
	}

	return "", false
}

// Refresh updates the cached autoscalingGroup, updating it's values and it's instances
func (a *AutoscalingGroupMonitor) refresh(autoscalingGroup *autoscaling.Group) error {

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

	a.autoscaling.desiredCapacity = *autoscalingGroup.DesiredCapacity

	for _, instance := range autoscalingGroup.Instances {
		_, ok := a.autoscaling.instanceMonitors[*instance.InstanceId]
		if !ok {
			log.Debugf("Found new instance to monitor in autoscaling %s: %s", a.autoscaling.autoscalingGroupName, *instance.InstanceId)
			instanceMonitor, err := newInstanceMonitor(a.awsConnection, a.autoscaling.autoscalingGroupName, *instance.InstanceId, a.deathNodeMark)
			if err != nil {
				log.Error(err)
				continue
			}
			a.autoscaling.instanceMonitors[*instance.InstanceId] = instanceMonitor
		}
	}

	var found bool
	for instanceID := range a.autoscaling.instanceMonitors {
		found = false
		for _, instance := range autoscalingGroup.Instances {
			if *instance.InstanceId == instanceID {
				found = true
				break
			}
		}
		if !found {
			log.Debugf("Instance %s has disappeared from ASG %s. Stop monitoring it", instanceID, a.autoscaling.autoscalingGroupName)
			delete(a.autoscaling.instanceMonitors, instanceID)
		}

	}

	return nil
}

// NumUndesiredInstances return the number of instances to be removed from the AutoscalingGroup
func (a *AutoscalingGroupMonitor) NumUndesiredInstances() int {

	if len(a.autoscaling.instanceMonitors)-len(a.getInstancesMarkedToBeRemoved()) > int(a.autoscaling.desiredCapacity) {
		return len(a.autoscaling.instanceMonitors) - int(a.autoscaling.desiredCapacity)
	}

	return 0
}

// RemoveInstance removes the instanceId from the AutoscalingGroupMonitor cache
func (a *AutoscalingGroupMonitor) RemoveInstance(instanceMonitor *InstanceMonitor) {
	delete(a.autoscaling.instanceMonitors, instanceMonitor.instance.instanceID)
}

// GetInstancesMarkedToBeRemoved return the instances in AutoscalingGroupMonitor cache that
// do have the deathnode mark
func (a *AutoscalingGroupMonitor) getInstancesMarkedToBeRemoved() []*InstanceMonitor {
	return a.getInstances(true)
}

// GetInstances return the instances in AutoscalingGroupMonitor cache that
// doesn't have the deathnode mark
func (a *AutoscalingGroupMonitor) GetInstances() []*InstanceMonitor {
	return a.getInstances(false)
}

func (a *AutoscalingGroupMonitor) getInstances(markedToBeRemoved bool) []*InstanceMonitor {

	instances := []*InstanceMonitor{}
	for _, instanceMonitor := range a.autoscaling.instanceMonitors {
		if instanceMonitor.instance.markedToBeRemoved == markedToBeRemoved {
			instances = append(instances, instanceMonitor)
		}
	}

	return instances
}
