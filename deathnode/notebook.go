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
// - remove instance protection
// - complete lifecycle action if there is no tasks running from the protected frameworks
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

		instanceMonitor, err := n.autoscalingGroups.GetInstanceByID(*instance.InstanceId)
		if err != nil {
			return err
		}

		// If the instance is protected, remove instance protection
		n.removeInstanceProtection(instanceMonitor)

		// Next iteration if an instance was previously deleted before delayDeleteSeconds
		if n.delayDeleteSeconds != 0 && time.Since(n.lastDeleteTimestamp).Seconds() < float64(n.delayDeleteSeconds) {
			log.Debugf("Seconds since last destroy: %v. No instances will be destroyed", time.Since(n.lastDeleteTimestamp).Seconds())
			continue
		}

		// If the instance have no tasks from protected frameworks, delete it
		hasFrameworks := n.mesosMonitor.HasProtectedFrameworksTasks(*instance.PrivateIpAddress)
		if !hasFrameworks {
			if instanceMonitor.GetLifecycleState() == "Terminating:Wait" {
				log.Infof("Destroy instance %s", *instanceMonitor.GetInstanceID())
				err := n.awsConnection.CompleteLifecycleAction(instanceMonitor.GetAutoscalingGroupID(), instanceMonitor.GetInstanceID())
				if err != nil {
					log.Errorf("Unable to complete lifecycle action on instance %s", *instance.InstanceId)
				}
				if n.delayDeleteSeconds != 0 {
					n.lastDeleteTimestamp = time.Now()
					continue
				}
			} else {
				log.Debugf("Instance %s waiting for AWS to start termination lifecycle", *instance.InstanceId)
			}
		} else {
			log.Debugf("Instance %s can't be deleted. It contains tasks from protected frameworks", *instance.InstanceId)
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
