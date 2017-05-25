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

	log.Debugf("Remove agent %s from autoscalingGroup", instance.GetIP())
	err := instance.RemoveFromAutoscalingGroup()
	if err != nil {
		return err
	}

	instanceRemoveRequest := &instanceRemoveRequest{
		instance: instance,
		time:     time.Now(),
	}

	n.instanceRemoveRequests[instance.GetIP()] = instanceRemoveRequest

	log.Debugf("Set agent %s in maintenance", instance.GetIP())
	err = n.setAgentsInMaintenance()
	if err != nil {
		return err
	}

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

func (n *Notebook) DestroyInstancesAttempt() error {

	for _, instanceRemoveRequest := range n.instanceRemoveRequests {
		log.Debugf("Check if instance %s has running tasks", instanceRemoveRequest.instance.GetIP())
		hasFrameworks := n.mesosMonitor.DoesSlaveHasFrameworks(instanceRemoveRequest.instance.GetIP())

		if !hasFrameworks {
			log.Infof("Destroying instance %s", instanceRemoveRequest.instance.GetIP())
			err := instanceRemoveRequest.instance.Destroy()
			if err != nil {
				log.Errorf("Error destroying instance %s", err)
				break
			}

			delete(n.instanceRemoveRequests, instanceRemoveRequest.instance.GetIP())
			err = n.setAgentsInMaintenance()
			if err != nil {
				log.Errorf("Error removing host %s from maintenance", err)
				break
			}
			log.Infof("Instance %s destroyed", instanceRemoveRequest.instance.GetIP())
		}
	}

	return nil
}
