package watchers

import "github.com/alanbover/deathnode/mesos"

type recommender interface {
	find(mesosAgents []mesos.AgentMonitor) *mesos.AgentMonitor
}

type firstAvailableAgent struct {}

func (c* firstAvailableAgent) find(mesosAgents []mesos.AgentMonitor) *mesos.AgentMonitor {
	return &mesosAgents[0]
}
