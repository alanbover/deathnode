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
	instance *aws.InstanceMonitor
	time     time.Time
}

type Notebook struct {
	mesosMonitor           *mesos.MesosMonitor
	instanceRemoveRequests map[string]*instanceRemoveRequest
}

func NewNotebook(mesosMonitor *mesos.MesosMonitor) *Notebook {
	return &Notebook{
		mesosMonitor:           mesosMonitor,
		instanceRemoveRequests: map[string]*instanceRemoveRequest{},
	}
}

func (n *Notebook) write(instance *aws.InstanceMonitor) error {

	log.Infof("Requesting to kill instance %s", instance.GetIP())


	err := n.setAgentsInMaintenance()
	if err != nil {
		return err
	}

	err = instance.RemoveFromAutoscalingGroup()
	if err != nil {
		return err
	}

	instanceRemoveRequest := &instanceRemoveRequest{
		instance: instance,
		time:     time.Now(),
	}

	n.instanceRemoveRequests[instance.GetIP()] = instanceRemoveRequest
	log.Debugf("Instance %s removed from ASG", instance.GetIP())

	return nil
}

func (n *Notebook) setAgentsInMaintenance() error {

	hosts := map[string]string{}
	for _, instanceRemoveRequest := range n.instanceRemoveRequests {
		agentInfo, _ := n.mesosMonitor.GetMesosSlaveByIp(instanceRemoveRequest.instance.GetIP())
		hosts[agentInfo.Hostname] = instanceRemoveRequest.instance.GetIP()
	}

	return n.mesosMonitor.SetMesosSlavesInMaintenance(hosts)
}

func (n *Notebook) KillAttempt() error {

	for _, instanceRemoveRequest := range n.instanceRemoveRequests {
		log.Debugf("Checking if instance %s can be killed", instanceRemoveRequest.instance.GetIP())
		hasFrameworks, err := n.mesosMonitor.DoesSlaveHasFrameworks(instanceRemoveRequest.instance.GetIP())
		if err != nil {
			log.Errorf("Instance %s can't be found in Mesos: ", instanceRemoveRequest.instance.GetIP())
		}
		if !hasFrameworks {
			log.Infof("Destroying instance %s", instanceRemoveRequest.instance.GetIP())
			instanceRemoveRequest.instance.Destroy()
			delete(n.instanceRemoveRequests, instanceRemoveRequest.instance.GetIP())
			err := n.setAgentsInMaintenance()
			if err != nil {
				return err
			}
		}
	}

	return nil
}
