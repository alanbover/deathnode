package deathnode

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
	"time"
)

type instanceRemoveRequest struct {
	instace *aws.InstanceMonitor
	time    time.Time
}

type Notebook struct {
	protectedFrameworks    []string
	mesosMonitor           *mesos.MesosMonitor
	instanceRemoveRequests []*instanceRemoveRequest
}

func NewNotebook(mesosMonitor *mesos.MesosMonitor, protectedFrameworks []string) *Notebook {
	return &Notebook{
		protectedFrameworks:    protectedFrameworks,
		mesosMonitor:           mesosMonitor,
		instanceRemoveRequests: []*instanceRemoveRequest{},
	}
}

func (n *Notebook) write(instance *aws.InstanceMonitor) error {

	log.Infof("Requesting to kill instance %s", instance.GetIP())

	instanceRemoveRequest := &instanceRemoveRequest{
		instace: instance,
		time:    time.Now(),
	}

	n.instanceRemoveRequests = append(n.instanceRemoveRequests, instanceRemoveRequest)
	instance.RemoveFromAutoscalingGroup()

	agentInfo, _ := n.mesosMonitor.GetMesosSlaveByIp(instance.GetIP())
	n.mesosMonitor.SetMesosSlaveInMaintenance(agentInfo.Hostname, instance.GetIP())

	return nil
}

func (n *Notebook) KillAttempt() error {

	for _, instanceRemoveRequest := range n.instanceRemoveRequests {
		log.Debugf("Checking if instance %s should be removed from ASG", instanceRemoveRequest.instace.GetIP())
		hasFrameworks, err := n.mesosMonitor.DoesSlaveHasFrameworks(
			instanceRemoveRequest.instace.GetIP(), n.protectedFrameworks)
		if err != nil {
			log.Errorf("Instance %s can't be found in Mesos: ", instanceRemoveRequest.instace.GetIP())
		}
		if !hasFrameworks {
			log.Infof("Removing instance %s from autoscaling group", instanceRemoveRequest.instace.GetIP())
			instanceRemoveRequest.instace.RemoveFromAutoscalingGroup()
		}
	}

	return nil
}
