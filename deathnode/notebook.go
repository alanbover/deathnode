package deathnode

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"time"
)

type Notebook struct {
	mesosMonitor        *mesos.MesosMonitor
	awsConnection       aws.AwsConnectionInterface
	autoscalingGroups   *aws.AutoscalingGroups
	delayDeleteSeconds  int
	lastDeleteTimestamp time.Time
}

func NewNotebook(autoscalingGroups *aws.AutoscalingGroups, awsConn aws.AwsConnectionInterface, mesosMonitor *mesos.MesosMonitor, delayDeleteSeconds int) *Notebook {

	return &Notebook{
		mesosMonitor:        mesosMonitor,
		awsConnection:       awsConn,
		autoscalingGroups:   autoscalingGroups,
		delayDeleteSeconds:  delayDeleteSeconds,
		lastDeleteTimestamp: time.Time{},
	}
}

func (n *Notebook) setAgentsInMaintenance(instances []*ec2.Instance) error {

	hosts := map[string]string{}
	for _, instance := range instances {
		hosts[*instance.PrivateDnsName] = *instance.PrivateIpAddress
	}

	return n.mesosMonitor.SetMesosSlavesInMaintenance(hosts)
}

func (n *Notebook) DestroyInstancesAttempt() error {

	// Get instances marked for removal
	instances, err := n.awsConnection.DescribeInstancesByTag(aws.DEATH_NODE_TAG_MARK)
	if err != nil {
		log.Debugf("Unable to find instances with %s tag", aws.DEATH_NODE_TAG_MARK)
		return err
	}

	// Set instances in maintenance
	n.setAgentsInMaintenance(instances)

	for _, instance := range instances {

		// If the instance belongs to an Autoscaling group, remove it
		autoscalingGroupName, found := n.autoscalingGroups.GetAutoscalingNameByInstanceId(*instance.InstanceId)
		if found {
			log.Infof("Remove instance %s from autoscaling %s", *instance.InstanceId, autoscalingGroupName)
			err := n.awsConnection.DetachInstance(autoscalingGroupName, *instance.InstanceId)
			if err != nil {
				log.Errorf("Unable to remove instance %s from autoscaling %s", *instance.InstanceId, autoscalingGroupName)
			}
		}

		// Exit if an instance was previously deleted before delayDeleteSeconds
		if n.delayDeleteSeconds != 0 && time.Since(n.lastDeleteTimestamp).Seconds() < float64(n.delayDeleteSeconds) {
			log.Debugf("Seconds since last destroy: %v. No instances will be destroyed", time.Since(n.lastDeleteTimestamp).Seconds())
			return nil
		}

		// If the instance have no tasks from protected frameworks, delete it
		hasFrameworks := n.mesosMonitor.DoesSlaveHasFrameworks(*instance.PrivateIpAddress)
		if !hasFrameworks {
			log.Infof("Destroy instance %s", *instance.InstanceId)
			err := n.awsConnection.TerminateInstance(*instance.InstanceId)
			if err != nil {
				log.Errorf("Unable to destroy instance %s", *instance.InstanceId)
			}
			if n.delayDeleteSeconds != 0 {
				n.lastDeleteTimestamp = time.Now()
				return nil
			}
		}
	}

	return nil
}
