package deathnode

// Given an autoscaling group, apply constraints to protect agents to be killed

import (
	"fmt"
	"github.com/alanbover/deathnode/monitor"
	"strings"
)

func newConstraint(constraint string) (constraint, error) {

	constraintType, constraintParams := func(constraint string) (string, string) {
		constraintSplit := strings.Split(constraint, "=")
		if len(constraintSplit) > 1 {
			return constraintSplit[0], constraintSplit[1]
		}
		return constraintSplit[0], ""
	}(constraint)

	switch constraintType {
	case "noContraint":
		return &noConstraint{}, nil
	case "protectedConstraint":
		return &protectedConstraint{}, nil
	case "filterFrameworkConstraint":
		return &filterFrameworkConstraint{constraintParams}, nil
	case "taskNameRegexpConstraint":
		return &taskNameRegexpConstraint{constraintParams}, nil
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

type filterFrameworkConstraint struct {
	framework string
}

func (c *filterFrameworkConstraint) filter(instanceMonitors []*monitor.InstanceMonitor, mesosMonitor *monitor.MesosMonitor) []*monitor.InstanceMonitor {

	filteredInstanceMonitors := []*monitor.InstanceMonitor{}
	for _, instanceMonitor := range instanceMonitors {
		if !mesosMonitor.HasFrameworks(instanceMonitor.IP(), c.framework) {
			filteredInstanceMonitors = append(filteredInstanceMonitors, instanceMonitor)
		}
	}

	if len(filteredInstanceMonitors) > 0 {
		return filteredInstanceMonitors
	}

	return instanceMonitors
}

type taskNameRegexpConstraint struct {
	regexp string
}

func (c *taskNameRegexpConstraint) filter(instanceMonitors []*monitor.InstanceMonitor, mesosMonitor *monitor.MesosMonitor) []*monitor.InstanceMonitor {

	filteredInstanceMonitors := []*monitor.InstanceMonitor{}
	for _, instanceMonitor := range instanceMonitors {
		if !mesosMonitor.HasTaskNameMatchRegexp(instanceMonitor.IP(), c.regexp) {
			filteredInstanceMonitors = append(filteredInstanceMonitors, instanceMonitor)
		}
	}

	if len(filteredInstanceMonitors) > 0 {
		return filteredInstanceMonitors
	}

	return instanceMonitors
}
