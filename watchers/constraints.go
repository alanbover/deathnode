package watchers

// Given an autoscaling group, apply constraints to protect agents to be killed

import "github.com/alanbover/deathnode/mesos"

type constrainst interface {
	filter(autoscalingGroupMonitor *mesos.AutoscalingGroupMonitor) []mesos.AgentMonitor
}

type noConstraint struct {}

func (c* noConstraint) filter(autoscalingGroupMonitor *mesos.AutoscalingGroupMonitor) []mesos.AgentMonitor {
	return *autoscalingGroupMonitor.GetAgents()
}
