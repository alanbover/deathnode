package deathnode

// Given an autoscaling group, apply constraints to protect agents to be killed

import (
	"fmt"
	"github.com/alanbover/deathnode/monitor"
)

func newConstraint(constraintType string) (constraint, error) {
	switch constraintType {
	case "noContraint":
		return &noConstraint{}, nil
	case "protectedConstraint":
		return &protectedConstraint{}, nil
	default:
		return nil, fmt.Errorf("Constraint type %v not found", constraintType)
	}
}

type constraint interface {
	filter([]*monitor.InstanceMonitor, *monitor.MesosMonitor) []*monitor.InstanceMonitor
}

type noConstraint struct{}

func (c *noConstraint) filter(instanceMonitors []*monitor.InstanceMonitor, mesosMonitor *monitor.MesosMonitor) []*monitor.InstanceMonitor {
	return instanceMonitors
}

type protectedConstraint struct{}

func (c *protectedConstraint) filter(instanceMonitors []*monitor.InstanceMonitor, mesosMonitor *monitor.MesosMonitor) []*monitor.InstanceMonitor {

	filteredInstanceMonitors := []*monitor.InstanceMonitor{}
	for _, instanceMonitor := range instanceMonitors {
		if !mesosMonitor.IsProtected(instanceMonitor.IP()) {
			filteredInstanceMonitors = append(filteredInstanceMonitors, instanceMonitor)
		}
	}

	if len(filteredInstanceMonitors) > 0 {
		return filteredInstanceMonitors
	}

	return instanceMonitors
}
