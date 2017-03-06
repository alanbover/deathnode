package watchers

// Stores the Mesos agents we want to kill. It will periodically review the state of the agents and kill them if
// they are not running any tasks

import "github.com/alanbover/deathnode/mesos"

type mesosAgentKillRequest struct {
	agent mesos.Agent
}

type Notebook struct {
	agentDeathRequests []mesosAgentKillRequest
}

func (n* Notebook) requestKillMesosAgent(agent *mesos.AgentMonitor) error {
	return nil
}
