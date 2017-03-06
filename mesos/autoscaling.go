package mesos

import (
	"github.com/alanbover/deathnode/aws"
)

type AutoscalingGroups []*AutoscalingGroupMonitor

type autoscalingGroup struct {
	autoscalingGroupName	string
	launchConfiguration	string
	desiredCapacity		int64
	maxSize			int64
	minSize 		int64
	agentMonitors		*[]AgentMonitor
}

type AutoscalingGroupMonitor struct {
	autoscaling 		*autoscalingGroup
	awsConnection		aws.ConnectionInterface
}

func NewAutoscalingGroups(conn aws.ConnectionInterface, autoscalingGroupNameList []string) (*AutoscalingGroups, error) {

	autoscalingGroups := new(AutoscalingGroups)

	for _, autoscalingGroupName := range autoscalingGroupNameList {

		autoscalingGroup, _ := NewAutoscalingGroupMonitor(conn, autoscalingGroupName)
		*autoscalingGroups = append(*autoscalingGroups, autoscalingGroup)
	}

	return autoscalingGroups, nil
}

func NewAutoscalingGroupMonitor(conn aws.ConnectionInterface, autoscalingGroupName string) (*AutoscalingGroupMonitor, error) {

	response, err := conn.DescribeAGByName(autoscalingGroupName)

	if err != nil {
		return &AutoscalingGroupMonitor{}, err
	}

	agentMonitors := []AgentMonitor{}

	for _, instance := range response.Instances {
		agentMonitor, _ := NewAgentMonitor(conn, autoscalingGroupName, *instance.LaunchConfigurationName, *instance.InstanceId)
		agentMonitors = append(agentMonitors, *agentMonitor)
	}

	return &AutoscalingGroupMonitor{
		autoscaling: &autoscalingGroup{
			autoscalingGroupName: autoscalingGroupName,
			launchConfiguration: *response.LaunchConfigurationName,
			desiredCapacity: *response.DesiredCapacity,
			maxSize: *response.MaxSize,
			minSize: *response.MinSize,
			agentMonitors: &agentMonitors,
		},
		awsConnection: conn,
	}, nil
}

func (a *AutoscalingGroupMonitor) Refresh() error {
	return nil
}

func (a *AutoscalingGroupMonitor) GetAgents() *[]AgentMonitor {
	return a.autoscaling.agentMonitors
}

func (a *AutoscalingGroupMonitor) NumUndesiredMesosAgents() int {
	if len(*a.autoscaling.agentMonitors) > int(a.autoscaling.desiredCapacity) {
		return int(a.autoscaling.desiredCapacity) - len(*a.autoscaling.agentMonitors)
	}

	return 0
}


