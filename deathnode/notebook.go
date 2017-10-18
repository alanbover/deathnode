package deathnode

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import (
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/monitor"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"time"
)

// Notebook stores the necessary information for deal with instances that should be deleted
type Notebook struct {
	mesosMonitor        *monitor.MesosMonitor
	autoscalingGroups   *monitor.AutoscalingServiceMonitor
	lastDeleteTimestamp time.Time
	ctx                 *context.ApplicationContext
}

// NewNotebook creates a notebook object, which is in charge of monitoring and delete instances marked to be deleted
func NewNotebook(ctx *context.ApplicationContext, autoscalingGroups *monitor.AutoscalingServiceMonitor,
	mesosMonitor *monitor.MesosMonitor) *Notebook {

	return &Notebook{
		mesosMonitor:        mesosMonitor,
		autoscalingGroups:   autoscalingGroups,
		lastDeleteTimestamp: time.Time{},
		ctx:                 ctx,
	}
}

func (n *Notebook) setAgentsInMaintenance(instances []*ec2.Instance) error {

	hosts := map[string]string{}
	for _, instance := range instances {
		hosts[*instance.PrivateDnsName] = *instance.PrivateIpAddress
	}

	return n.mesosMonitor.SetMesosAgentsInMaintenance(hosts)
}

func (n *Notebook) shouldWaitForNextDestroy() bool {
	return time.Since(n.lastDeleteTimestamp).Seconds() <= float64(n.ctx.Conf.DelayDeleteSeconds)
}

func (n *Notebook) destroyInstance(instanceMonitor *monitor.InstanceMonitor) error {

	if instanceMonitor.LifecycleState() == monitor.LifecycleStateTerminatingWait {
		log.Infof("Destroy instance %s", *instanceMonitor.InstanceID())
		err := n.ctx.AwsConn.CompleteLifecycleAction(
			instanceMonitor.AutoscalingGroupID(), instanceMonitor.InstanceID())
		if err != nil {
			log.Errorf("Unable to complete lifecycle action on instance %s", *instanceMonitor.InstanceID())
			return err
		}
		if n.ctx.Conf.DelayDeleteSeconds != 0 {
			n.lastDeleteTimestamp = time.Now()
		}
	} else {
		log.Debugf("Instance %s waiting for AWS to start termination lifecycle", *instanceMonitor.InstanceID())
	}
	return nil
}

func (n *Notebook) destroyInstanceAttempt(instance *ec2.Instance) error {

	log.Debugf("Starting process to delete instance %s", *instance.InstanceId)

	instanceMonitor, err := n.autoscalingGroups.GetInstanceByID(*instance.InstanceId)
	if err != nil {
		return err
	}

	// If the instance is protected, remove instance protection
	n.removeInstanceProtection(instanceMonitor)

	// Check if we need to wait before destroy another instance
	if n.shouldWaitForNextDestroy() {
		log.Debugf("Seconds since last destroy: %v. Instance %s will not be destroyed",
			time.Since(n.lastDeleteTimestamp).Seconds(), *instance.InstanceId)
		return nil
	}

	// If the instance can be killed, delete it
	if !n.mesosMonitor.IsProtected(*instance.PrivateIpAddress) {
		if err := n.destroyInstance(instanceMonitor); err != nil {
			return err
		}
	}
	return nil
}

// DestroyInstancesAttempt iterates around all instances marked to be deleted, and:
// - set them in maintenance
// - remove instance protection
// - complete lifecycle action if there is no tasks running from the protected frameworks
func (n *Notebook) DestroyInstancesAttempt() error {

	// Get instances marked for removal
	instances, err := n.ctx.AwsConn.DescribeInstancesByTag(n.ctx.Conf.DeathNodeMark)
	if err != nil {
		log.Debugf("Error retrieving instances with tag %s", n.ctx.Conf.DeathNodeMark)
		return err
	}

	// Set instances in maintenance
	n.setAgentsInMaintenance(instances)

	for _, instance := range instances {
		if err := n.destroyInstanceAttempt(instance); err != nil {
			log.Warn(err)
		}
	}

	return nil
}

func (n *Notebook) removeInstanceProtection(instance *monitor.InstanceMonitor) error {

	if instance.IsProtected() {
		return instance.RemoveInstanceProtection()
	}
	return nil
}
