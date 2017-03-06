package mesos

import (
	"testing"
	"github.com/alanbover/deathnode/aws/mock"
)

func TestNewAutoscalingGroupMonitor(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName": &[]string{"default"},
		},
	}

	monitor, _ := NewAutoscalingGroupMonitor(conn, "some-Autoscaling-Group")

	if monitor == nil {
		t.Fatal("TestNewAutoscalingGroupMonitor return nil")
	}

	if (*monitor.autoscaling.agentMonitors)[0].agent.instanceId == (*monitor.autoscaling.agentMonitors)[1].agent.instanceId {
		t.Fatal("Agent 0 has same instanceId than agent 1")
	}

	if monitor.NumUndesiredMesosAgents() > 0 {
		t.Fatal("Incorrect number of undesired mesos agents")
	}
}

func NumUndesiredMesosAgents(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName": &[]string{"one_undesired_host"},
		},
	}

	monitor, _ := NewAutoscalingGroupMonitor(conn, "some-Autoscaling-Group")

	if monitor.NumUndesiredMesosAgents() != 1 {
		t.Fatal("Incorrect number of undesired mesos agents")
	}
}

func TestNewAutoscalingGroups(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName": &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(conn, autoscalingGroupNames)

	if (*autoscalingGroups)[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" {
		t.Fatal("Error creating AutoscalingGroups")
	}
}

func TestRemoveAgentFromAutoscalingGroup(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName": &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(conn, autoscalingGroupNames)

	agentMonitors := (*autoscalingGroups)[0].autoscaling.agentMonitors
	(*agentMonitors)[0].RemoveFromAutoscalingGroup()

	callArguments := conn.Requests["DetachInstance"]

	if (*callArguments)[0] != "some-Autoscaling-Group" || (*callArguments)[1] != "i-34719eb8" {
		t.Fatal("Incorrect parameters when calling RemoveFromAutoscalingGroup")
	}
}
