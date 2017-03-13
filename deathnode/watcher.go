package deathnode

// Given an autoscaling group, decides which is/are the best agent/s to kill

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
)

type DeathNodeWatcher struct {
	notebook     *Notebook
	mesosMonitor *mesos.MesosMonitor
	constraints  constraint
	recommender  recommender
}

func NewDeathNodeWatcher(notebook *Notebook, mesosMonitor *mesos.MesosMonitor, constraintType, recommenderType string) *DeathNodeWatcher {

	contrainsts, err := newConstraint(constraintType)
	if err != nil {
		log.Fatal(err)
	}

	recommender, err := newRecommender(recommenderType)
	if err != nil {
		log.Fatal(err)
	}

	return &DeathNodeWatcher{
		notebook:     notebook,
		mesosMonitor: mesosMonitor,
		constraints:  contrainsts,
		recommender:  recommender,
	}
}

func (y *DeathNodeWatcher) CheckIfInstancesToKill(autoscalingMonitor *aws.AutoscalingGroupMonitor) error {

	autoscalingMonitor.Refresh()
	numUndesiredMesosAgents := autoscalingMonitor.NumUndesiredInstances()

	removedAgents := 0

	for removedAgents < numUndesiredMesosAgents {
		allowedAgentsToKill := y.constraints.filter(autoscalingMonitor)
		bestAgentToKill := y.recommender.find(allowedAgentsToKill)
		y.notebook.write(bestAgentToKill)
		removedAgents += 1
	}

	return nil
}
