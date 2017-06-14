package deathnode

// Given an autoscaling group, apply constraints to protect agents to be killed

import (
	"fmt"
	"github.com/alanbover/deathnode/aws"
)

func newConstraint(constraintType string) (constraint, error) {
	switch constraintType {
	case "noContraint":
		return &noConstraint{}, nil
	default:
		return nil, fmt.Errorf("Contraint type %v not found", constraintType)
	}
}

type constraint interface {
	filter([]*aws.InstanceMonitor) []*aws.InstanceMonitor
}

type noConstraint struct{}

func (c *noConstraint) filter(instanceMonitors []*aws.InstanceMonitor) []*aws.InstanceMonitor {
	return instanceMonitors
}
