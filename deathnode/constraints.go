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
	default:
		return nil, fmt.Errorf("Constraint type %v not found", constraintType)
	}
}

type constraint interface {
	filter([]*monitor.InstanceMonitor) []*monitor.InstanceMonitor
}

type noConstraint struct{}

func (c *noConstraint) filter(instanceMonitors []*monitor.InstanceMonitor) []*monitor.InstanceMonitor {
	return instanceMonitors
}
