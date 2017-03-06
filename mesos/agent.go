package mesos

import "github.com/alanbover/deathnode/aws"

type task struct {
	name           string
	start_time     string
	framework_name string
}

type Agent struct {
	autoscalingGroupId  string
	launchConfiguration string
	ipAddress           string
	instanceId          string
	runningTasks        *[]task
	inMaintenance       bool
	isProtected         bool
}

type AgentMonitor struct {
	agent         *Agent
	awsConnection aws.ConnectionInterface
}

func NewAgentMonitor(conn aws.ConnectionInterface, autoscalingGroupId, launchConfiguration, instanceId string) (*AgentMonitor, error) {

	response, err := conn.DescribeInstanceById(instanceId)

	if err != nil {
		return &AgentMonitor{}, err
	}

	return &AgentMonitor{
		agent: &Agent{
			autoscalingGroupId:  autoscalingGroupId,
			launchConfiguration: launchConfiguration,
			ipAddress:           *response.PrivateIpAddress,
			instanceId:          instanceId,
			runningTasks:        &[]task{},
			inMaintenance:       false,
			isProtected:         true,
		},
		awsConnection: conn,
	}, nil
}

func (a *AgentMonitor) IsDrained(registeredFrameworks *[]string) (bool, error) {

	for _, registeredFramework := range *registeredFrameworks {
		for _, task := range *a.agent.runningTasks {
			if registeredFramework == task.framework_name {
				return false, nil
			}
		}
	}

	return true, nil
}

func (a *AgentMonitor) IsInMaintenance() bool {
	return a.agent.inMaintenance
}

func (a *AgentMonitor) SetInMaintenance() error {
	return nil
}

func (a *AgentMonitor) GetNumberTasks() int {
	return len(*a.agent.runningTasks)
}

func (a *AgentMonitor) Destroy() error {
	err := a.awsConnection.TerminateInstance(a.agent.instanceId)
	return err
}

func (a *AgentMonitor) RemoveFromAutoscalingGroup() error {
	err := a.awsConnection.DetachInstance(a.agent.autoscalingGroupId, a.agent.instanceId)
	return err
}
