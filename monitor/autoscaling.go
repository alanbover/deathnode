package monitor

import (
	"fmt"
	"github.com/alanbover/deathnode/context"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	log "github.com/sirupsen/logrus"
)

// AutoscalingGroupMonitorCollection holds a map of [ASGprefix][ASGname]AutoscalingGroupMonitor
type AutoscalingGroupMonitorCollection struct {
	monitors map[string]map[string]*AutoscalingGroupMonitor
	context  *context.ApplicationContext
}

// AutoscalingGroupMonitor monitors an AWS autoscaling group, caching it's data
type AutoscalingGroupMonitor struct {
	autoscaling *autoscalingGroup
	context     *context.ApplicationContext
}

type autoscalingGroup struct {
	autoscalingGroupName string
	desiredCapacity      int64
	instanceMonitors     map[string]*InstanceMonitor
}

var lifeCycleTimeout int64 = 3600

// NewAutoscalingGroupMonitors returns an AutoscalingGroups object
func NewAutoscalingGroupMonitors(ctx *context.ApplicationContext) *AutoscalingGroupMonitorCollection {

	monitors := map[string]map[string]*AutoscalingGroupMonitor{}
	for _, autoscalingGroupName := range ctx.Conf.AutoscalingGroupPrefixes {
		monitors[autoscalingGroupName] = map[string]*AutoscalingGroupMonitor{}
	}

	autoscalingGroups := &AutoscalingGroupMonitorCollection{
		monitors: monitors,
		context:  ctx,
	}

	return autoscalingGroups
}

// NewAutoscalingGroupMonitor returns a "empty" AutoscalingGroupMonitor object
func newAutoscalingGroupMonitor(context *context.ApplicationContext, autoscalingGroupName string) (*AutoscalingGroupMonitor, error) {

	return &AutoscalingGroupMonitor{
		autoscaling: &autoscalingGroup{
			autoscalingGroupName: autoscalingGroupName,
			desiredCapacity:      0,
			instanceMonitors:     map[string]*InstanceMonitor{},
		},
		context: context,
	}, nil
}

// GetInstanceByID returns the instanceMonitor related with the instanceId
func (a *AutoscalingGroupMonitorCollection) GetInstanceByID(instanceID string) (*InstanceMonitor, error) {

	for _, autoscalingPrefix := range a.monitors {
		for _, autoscalingMonitor := range autoscalingPrefix {
			if instance, ok := autoscalingMonitor.autoscaling.instanceMonitors[instanceID]; ok {
				return instance, nil
			}
		}
	}
	return nil, fmt.Errorf("InstanceId %s not found", instanceID)
}

// GetAutoscalingGroupMonitors returns all AutoscalingGroupMonitors cached in AutoscalingGroups
func (a *AutoscalingGroupMonitorCollection) GetAutoscalingGroupMonitors() []*AutoscalingGroupMonitor {

	var monitors = []*AutoscalingGroupMonitor{}

	for autoscalingGroupPrefix := range a.monitors {
		for autoscalingGroupName := range a.monitors[autoscalingGroupPrefix] {
			monitors = append(monitors, a.monitors[autoscalingGroupPrefix][autoscalingGroupName])
		}
	}

	return monitors
}

func findAutoscalingGroupInResponse(autoscalingGroupName string, response []*autoscaling.Group) (*autoscaling.Group, bool) {

	for _, autoscalingGroup := range response {
		if autoscalingGroupName == *autoscalingGroup.AutoScalingGroupName {
			return autoscalingGroup, true
		}
	}

	return nil, false
}

// Refresh updates autoscalingGroups caching all AWS autoscaling groups given the N prefixes
// provided when AutoscalingGroups was created
func (a *AutoscalingGroupMonitorCollection) Refresh() error {

	for autoscalingGroupPrefix := range a.monitors {

		response, err := a.context.AwsConn.DescribeAGByName(autoscalingGroupPrefix)
		if err != nil {
			return err
		}
		if len(response) == 0 {
			log.Warnf("No autoscaling groups found under autoscalingGroupPrefix %s", autoscalingGroupPrefix)
		}

		// find new autoscalingGroups
		for _, autoscalingGroupResponse := range response {
			if _, ok := a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName]; !ok {
				a.newAutoscalingGroup(autoscalingGroupPrefix, autoscalingGroupResponse)
			}
		}

		for autoscalingGroupName := range a.monitors[autoscalingGroupPrefix] {
			if autoscalingGroup, ok := findAutoscalingGroupInResponse(autoscalingGroupName, response); ok {
				a.monitors[autoscalingGroupPrefix][autoscalingGroupName].refresh(autoscalingGroup)
			} else {
				log.Infof("Autoscaling group %s removed. Deleting it", autoscalingGroupName)
				delete(a.monitors[autoscalingGroupPrefix], autoscalingGroupName)
			}
		}
	}
	return nil
}

func (a *AutoscalingGroupMonitorCollection) newAutoscalingGroup(autoscalingGroupPrefix string, autoscalingGroupResponse *autoscaling.Group) {

	log.Infof("Found new autoscalingGroup to monitor: %s", *autoscalingGroupResponse.AutoScalingGroupName)
	autoscalingGroupMonitor, _ := newAutoscalingGroupMonitor(a.context, *autoscalingGroupResponse.AutoScalingGroupName)

	ok, _ := a.context.AwsConn.HasLifeCycleHook(autoscalingGroupResponse.AutoScalingGroupName)
	if !ok {
		log.Infof("Setting lifecyclehook for autoscaling %s", *autoscalingGroupResponse.AutoScalingGroupName)
		err := a.context.AwsConn.PutLifeCycleHook(autoscalingGroupResponse.AutoScalingGroupName, &lifeCycleTimeout)
		if err != nil {
			log.Warnf("Error putting lifecyclehook to autoscaling %s: %s", *autoscalingGroupResponse.AutoScalingGroupName, err)
		}
	} else {
		log.Infof("Autoscaling %s already have set lifecyclehook. Ignoring it...", *autoscalingGroupResponse.AutoScalingGroupName)
	}

	a.monitors[autoscalingGroupPrefix][*autoscalingGroupResponse.AutoScalingGroupName] = autoscalingGroupMonitor
}

// GetNumUndesiredInstances return the number of instances to be removed from the AutoscalingGroup
func (a *AutoscalingGroupMonitor) GetNumUndesiredInstances() int {

	if len(a.autoscaling.instanceMonitors)-len(a.getInstancesMarkedToBeRemoved()) > int(a.autoscaling.desiredCapacity) {
		return len(a.autoscaling.instanceMonitors) - int(a.autoscaling.desiredCapacity)
	}

	return 0
}

// GetInstances return the instances in AutoscalingGroupMonitor cache that
// doesn't have the deathnode mark
func (a *AutoscalingGroupMonitor) GetInstances() []*InstanceMonitor {
	return a.getInstances(false)
}

func (a *AutoscalingGroupMonitor) refresh(autoscalingGroup *autoscaling.Group) error {

	if err := a.enforceInstanceProtection(autoscalingGroup); err != nil {
		return err
	}

	a.autoscaling.desiredCapacity = *autoscalingGroup.DesiredCapacity

	// find new instances in autoscaling group
	for _, instance := range autoscalingGroup.Instances {
		_, ok := a.autoscaling.instanceMonitors[*instance.InstanceId]
		if !ok {
			if err := a.newInstance(autoscalingGroup, instance); err != nil {
				log.Error(err)
				continue
			}
		}
	}

	for instanceID := range a.autoscaling.instanceMonitors {
		if instance, ok := findInstanceInResponse(instanceID, autoscalingGroup); ok {
			a.autoscaling.instanceMonitors[*instance.InstanceId].setLifecycleState(*instance.LifecycleState)
		} else {
			log.Debugf("Instance %s has disappeared from ASG %s. Stop monitoring it", instanceID, a.autoscaling.autoscalingGroupName)
			delete(a.autoscaling.instanceMonitors, instanceID)
		}
	}

	return nil
}

func (a *AutoscalingGroupMonitor) setInstanceProtection(autoscalingGroup *autoscaling.Group) error {

	log.Infof("Setting autoscaling %s and it's instances scaleInProtection flag", *autoscalingGroup.AutoScalingGroupName)
	instancesToProtect := []*string{}

	for _, instance := range autoscalingGroup.Instances {
		instancesToProtect = append(instancesToProtect, instance.InstanceId)
	}

	err := a.context.AwsConn.SetASGInstanceProtection(autoscalingGroup.AutoScalingGroupName, instancesToProtect)
	if err != nil {
		return err
	}
	return nil
}

func (a *AutoscalingGroupMonitor) enforceInstanceProtection(autoscalingGroup *autoscaling.Group) error {

	if !*autoscalingGroup.NewInstancesProtectedFromScaleIn {
		if err := a.setInstanceProtection(autoscalingGroup); err != nil {
			return err
		}
	}

	log.Debugf("Autoscaling %s already has scaleInProtection set. Ignoring it...", *autoscalingGroup.AutoScalingGroupName)
	return nil
}

func (a *AutoscalingGroupMonitor) newInstance(autoscalingGroup *autoscaling.Group, instance *autoscaling.Instance) error {

	log.Debugf("Found new instance to monitor in autoscaling %s: %s", a.autoscaling.autoscalingGroupName, *instance.InstanceId)
	instanceMonitor, err := newInstanceMonitor(
		a.context.AwsConn, a.autoscaling.autoscalingGroupName, *instance.InstanceId, a.context.Conf.DeathNodeMark, *instance.LifecycleState, true,
	)
	if err != nil {
		return err
	}

	a.autoscaling.instanceMonitors[*instance.InstanceId] = instanceMonitor
	return nil
}

func findInstanceInResponse(instanceID string, autoscalingGroup *autoscaling.Group) (*autoscaling.Instance, bool) {

	for _, instance := range autoscalingGroup.Instances {
		if *instance.InstanceId == instanceID {
			return instance, true
		}
	}

	return nil, false
}

// GetInstancesMarkedToBeRemoved return the instances in AutoscalingGroupMonitor cache that
// do have the deathnode mark
func (a *AutoscalingGroupMonitor) getInstancesMarkedToBeRemoved() []*InstanceMonitor {
	return a.getInstances(true)
}

func (a *AutoscalingGroupMonitor) getInstances(markedToBeRemoved bool) []*InstanceMonitor {

	instances := []*InstanceMonitor{}
	for _, instanceMonitor := range a.autoscaling.instanceMonitors {
		if instanceMonitor.instance.isMarkedToBeRemoved == markedToBeRemoved {
			instances = append(instances, instanceMonitor)
		}
	}

	return instances
}
