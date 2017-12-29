package monitor

import (
	"fmt"
	"github.com/alanbover/deathnode/context"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	log "github.com/sirupsen/logrus"
)

// AutoscalingServiceMonitor holds a map of [ASGprefix][ASGname]AutoscalingGroupMonitor
type AutoscalingServiceMonitor struct {
	autoscalingMonitors map[string]map[string]*AutoscalingGroupMonitor
	ctx                 *context.ApplicationContext
}

// AutoscalingGroupMonitor monitors an AWS autoscaling group, caching it's data
type AutoscalingGroupMonitor struct {
	autoscalingGroupName string
	desiredCapacity      int64
	instanceMonitors     map[string]*InstanceMonitor
	ctx                  *context.ApplicationContext
}

// LifeCycleTimeout sets the time set for LifeCycleHooks timeout
var LifeCycleTimeout int64 = 3600

// NewAutoscalingServiceMonitor returns an AutoscalingServiceMonitor object
func NewAutoscalingServiceMonitor(ctx *context.ApplicationContext) *AutoscalingServiceMonitor {

	autoscalingMonitors := map[string]map[string]*AutoscalingGroupMonitor{}
	for _, autoscalingGroupPrefix := range ctx.Conf.AutoscalingGroupPrefixes {
		autoscalingMonitors[autoscalingGroupPrefix] = map[string]*AutoscalingGroupMonitor{}
	}

	autoscalingServiceMonitor := &AutoscalingServiceMonitor{
		autoscalingMonitors: autoscalingMonitors,
		ctx:                 ctx,
	}

	return autoscalingServiceMonitor
}

// NewAutoscalingGroupMonitor returns a "empty" AutoscalingGroupMonitor object
func newAutoscalingGroupMonitor(ctx *context.ApplicationContext,
	autoscalingGroupName string) (*AutoscalingGroupMonitor, error) {

	return &AutoscalingGroupMonitor{
		autoscalingGroupName: autoscalingGroupName,
		desiredCapacity:      0,
		instanceMonitors:     map[string]*InstanceMonitor{},
		ctx:                  ctx,
	}, nil
}

// GetInstanceByID returns the instanceMonitor related with the instanceId
func (a *AutoscalingServiceMonitor) GetInstanceByID(instanceID string) (*InstanceMonitor, error) {

	for _, autoscalingPrefix := range a.autoscalingMonitors {
		for _, autoscalingMonitor := range autoscalingPrefix {
			if instance, ok := autoscalingMonitor.instanceMonitors[instanceID]; ok {
				return instance, nil
			}
		}
	}
	return nil, fmt.Errorf("InstanceId %s not found", instanceID)
}

// GetAutoscalingGroupMonitorsList returns all AutoscalingGroupMonitors cached in AutoscalingGroups in a list
func (a *AutoscalingServiceMonitor) GetAutoscalingGroupMonitorsList() []*AutoscalingGroupMonitor {

	var monitors = []*AutoscalingGroupMonitor{}

	for autoscalingGroupPrefix := range a.autoscalingMonitors {
		for autoscalingGroupName := range a.autoscalingMonitors[autoscalingGroupPrefix] {
			monitors = append(monitors, a.autoscalingMonitors[autoscalingGroupPrefix][autoscalingGroupName])
		}
	}

	return monitors
}

func findAutoscalingGroup(autoscalingGroupName string,
	response []*autoscaling.Group) (*autoscaling.Group, bool) {

	for _, autoscalingGroup := range response {
		if autoscalingGroupName == *autoscalingGroup.AutoScalingGroupName {
			return autoscalingGroup, true
		}
	}

	return nil, false
}

// Refresh updates autoscalingGroups caching all AWS autoscaling groups given the N prefixes
// provided when AutoscalingGroups was created
func (a *AutoscalingServiceMonitor) Refresh() error {

	for autoscalingGroupPrefix := range a.autoscalingMonitors {
		if err := a.refreshAutoscalingPrefix(autoscalingGroupPrefix); err != nil {
			log.Warning(err)
		}
	}
	return nil
}

func (a *AutoscalingServiceMonitor) refreshAutoscalingPrefix(prefix string) error {

	response, err := a.ctx.AwsConn.DescribeAGsByPrefix(prefix)
	if err != nil {
		return err
	}
	if len(response) == 0 {
		log.Warnf("No autoscaling groups found under autoscalingGroupPrefix %s",
			prefix)
	}

	// find new autoscalingGroups
	for _, autoscalingGroup := range response {
		if _, ok := a.autoscalingMonitors[prefix][*autoscalingGroup.AutoScalingGroupName]; !ok {
			a.newAutoscalingGroupMonitor(prefix, *autoscalingGroup.AutoScalingGroupName)
		}
	}

	for autoscalingGroupName := range a.autoscalingMonitors[prefix] {
		if autoscalingGroup, ok := findAutoscalingGroup(autoscalingGroupName, response); ok {
			a.autoscalingMonitors[prefix][autoscalingGroupName].refresh(autoscalingGroup)
		} else {
			log.Infof("Autoscaling group %s removed. Deleting it", autoscalingGroupName)
			delete(a.autoscalingMonitors[prefix], autoscalingGroupName)
		}
	}

	return nil
}

func (a *AutoscalingServiceMonitor) newAutoscalingGroupMonitor(autoscalingGroupPrefix string,
	autoscalingGroupName string) {

	log.Infof("Found new autoscalingGroup to monitor: %s", autoscalingGroupName)
	autoscalingGroupMonitor, _ := newAutoscalingGroupMonitor(a.ctx, autoscalingGroupName)

	// Set life cycle hook if it's not set already
	ok, _ := a.ctx.AwsConn.HasLifeCycleHook(autoscalingGroupName)
	if !ok {
		log.Infof("Setting lifecyclehook for autoscaling %s", autoscalingGroupName)
		err := a.ctx.AwsConn.PutLifeCycleHook(autoscalingGroupName, &LifeCycleTimeout)
		if err != nil {
			log.Warnf("Error putting lifecyclehook to autoscaling %s: %s",
				autoscalingGroupName, err)
			return
		}
	} else {
		log.Infof("Autoscaling %s already have set lifecyclehook. Ignoring it...",
			autoscalingGroupName)
	}

	a.autoscalingMonitors[autoscalingGroupPrefix][autoscalingGroupName] = autoscalingGroupMonitor
}

// GetNumUndesiredInstances return the number of instances to be removed from the AutoscalingGroup
func (a *AutoscalingGroupMonitor) GetNumUndesiredInstances() int {

	activeInstances := len(a.instanceMonitors) - len(a.getInstancesMarkedToBeRemoved())
	if activeInstances > int(a.desiredCapacity) {
		return len(a.instanceMonitors) - int(a.desiredCapacity)
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

	a.desiredCapacity = *autoscalingGroup.DesiredCapacity

	// find new instances in autoscaling group
	for _, instance := range autoscalingGroup.Instances {
		_, ok := a.instanceMonitors[*instance.InstanceId]
		if !ok {
			if err := a.newInstance(instance); err != nil {
				log.Error(err)
				continue
			}
		}
	}

	for instanceID := range a.instanceMonitors {
		if instance, ok := findInstance(instanceID, autoscalingGroup); ok {
			a.instanceMonitors[*instance.InstanceId].setLifecycleState(*instance.LifecycleState)
		} else {
			log.Debugf("Instance %s has disappeared from ASG %s. Stop monitoring it",
				instanceID, a.autoscalingGroupName)
			delete(a.instanceMonitors, instanceID)
		}
	}

	return nil
}

func (a *AutoscalingGroupMonitor) setInstanceProtection(autoscalingGroup *autoscaling.Group) error {

	log.Infof("Setting autoscaling %s and it's instances scaleInProtection flag",
		*autoscalingGroup.AutoScalingGroupName)

	instancesToProtect := []*string{}
	for _, instance := range autoscalingGroup.Instances {
		instancesToProtect = append(instancesToProtect, instance.InstanceId)
	}

	err := a.ctx.AwsConn.SetASGInstanceProtection(autoscalingGroup.AutoScalingGroupName, instancesToProtect)
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

func (a *AutoscalingGroupMonitor) newInstance(instance *autoscaling.Instance) error {

	log.Debugf("Found new instance to monitor in autoscaling %s: %s",
		a.autoscalingGroupName, *instance.InstanceId)

	instanceMonitor, err := newInstanceMonitor(
		a.ctx, a.autoscalingGroupName, *instance.InstanceId, *instance.LifecycleState, true)
	if err != nil {
		return err
	}

	a.instanceMonitors[*instance.InstanceId] = instanceMonitor
	return nil
}

func findInstance(instanceID string, autoscalingGroup *autoscaling.Group) (*autoscaling.Instance, bool) {

	for _, instance := range autoscalingGroup.Instances {
		if *instance.InstanceId == instanceID {
			return instance, true
		}
	}

	return nil, false
}

func (a *AutoscalingGroupMonitor) getInstancesMarkedToBeRemoved() []*InstanceMonitor {
	return a.getInstances(true)
}

func (a *AutoscalingGroupMonitor) getInstances(markedToBeRemoved bool) []*InstanceMonitor {

	instances := []*InstanceMonitor{}
	for _, instanceMonitor := range a.instanceMonitors {
		if instanceMonitor.IsMarkedToBeRemoved() == markedToBeRemoved {
			instances = append(instances, instanceMonitor)
		}
	}

	return instances
}
