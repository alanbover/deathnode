package watchers

// Given an autoscaling group, decides which is/are the best agent/s to kill

import "github.com/alanbover/deathnode/mesos"

type MesosAutoscalingWatcher struct {
	notebook *Notebook
	constraints constrainst
	recommender recommender
}

func NewMesosAutoscalingWatcher(registeredFrameworks []string) *MesosAutoscalingWatcher {
	notebook := &Notebook{}

	return &MesosAutoscalingWatcher{
		notebook: notebook,
		constraints: &noConstraint{},
		recommender: &firstAvailableAgent{},
	}
}

func (y* MesosAutoscalingWatcher) CheckIfInstancesToKill(autoscalingMonitor *mesos.AutoscalingGroupMonitor) error {

	autoscalingMonitor.Refresh()
	numUndesiredMesosAgents := autoscalingMonitor.NumUndesiredMesosAgents()

	removedAgents := 0

	for removedAgents < numUndesiredMesosAgents {
		allowedAgents := y.constraints.filter(autoscalingMonitor)
		bestAgentToKill := y.recommender.find(allowedAgents)
		y.notebook.requestKillMesosAgent(bestAgentToKill)
		removedAgents += 1
	}

	return nil
}
