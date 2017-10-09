package deathnode

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/monitor"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"time"
)

// Notebook stores the necessary information for deal with instances that should be deleted
type Notebook struct {
	mesosMonitor        *monitor.MesosMonitor
	awsConnection       aws.ClientInterface
	autoscalingGroups   *monitor.AutoscalingGroupsMonitor
	delayDeleteSeconds  int
	lastDeleteTimestamp time.Time
	deathNodeMark       string
}

// NewNotebook creates a notebook object, which is in charge of monitoring and delete instances marked to be deleted
func NewNotebook(autoscalingGroups *monitor.AutoscalingGroupsMonitor, awsConn aws.ClientInterface, mesosMonitor *monitor.MesosMonitor, delayDeleteSeconds int, deathNodeMark string) *Notebook {

	return &Notebook{
		mesosMonitor:        mesosMonitor,
		awsConnection:       awsConn,
		autoscalingGroups:   autoscalingGroups,
		delayDeleteSeconds:  delayDeleteSeconds,
		lastDeleteTimestamp: time.Time{},
		deathNodeMark:       deathNodeMark,
	}
}

func (n *Notebook) setAgentsInMaintenance(instances []*ec2.Instance) error {

	hosts := map[string]string{}
	for _, instance := range instances {
		hosts[*instance.PrivateDnsName] = *instance.PrivateIpAddress
	}

	return n.mesosMonitor.SetMesosAgentsInMaintenance(hosts)
}

// DestroyInstancesAttempt iterates around all instances marked to be deleted, and:
// - set them in maintenance
// - remove them from it's ASG
// - remove the instance if there is no tasks running from the protected frameworks
func (n *Notebook) DestroyInstancesAttempt() error {

	// Get instances marked for removal
	instances, err := n.awsConnection.DescribeInstancesByTag(n.deathNodeMark)
	if err != nil {
		log.Debugf("Unable to find instances with %s tag", n.deathNodeMark)
		return err
	}

	// Set instances in maintenance
	n.setAgentsInMaintenance(instances)

	for _, instance := range instances {

		log.Debugf("Starting process to delete instance %s", *instance.InstanceId)
		// If the instance belongs to an Autoscaling group, remove it
		autoscalingGroupName, found := n.autoscalingGroups.GetAutoscalingNameByInstanceID(*instance.InstanceId)
		if found {
			log.Infof("Remove instance %s from autoscaling %s", *instance.InstanceId, autoscalingGroupName)
			err := n.awsConnection.DetachInstance(autoscalingGroupName, *instance.InstanceId)
			if err != nil {
				log.Errorf("Unable to remove instance %s from autoscaling %s", *instance.InstanceId, autoscalingGroupName)
			}
		}

		// Next iteration if an instance was previously deleted before delayDeleteSeconds
		if n.delayDeleteSeconds != 0 && time.Since(n.lastDeleteTimestamp).Seconds() < float64(n.delayDeleteSeconds) {
			log.Debugf("Seconds since last destroy: %v. No instances will be destroyed", time.Since(n.lastDeleteTimestamp).Seconds())
			continue
		}

		// If the instance can be killed, delete it
		isProtected, message := n.mesosMonitor.IsProtected(*instance.PrivateIpAddress)

		if !isProtected {
			log.Infof("Destroy instance %s", *instance.InstanceId)
			err := n.awsConnection.TerminateInstance(*instance.InstanceId)
			if err != nil {
				log.Errorf("Unable to destroy instance %s", *instance.InstanceId)
			}
			if n.delayDeleteSeconds != 0 {
				n.lastDeleteTimestamp = time.Now()
				continue
			}
		} else {
			log.Debugf("Instance %s can't be deleted. %s", *instance.InstanceId, message)
		}
	}

	return nil
}
