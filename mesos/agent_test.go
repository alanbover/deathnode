package mesos

import (
	"testing"
	"github.com/alanbover/deathnode/aws/mock"
)

func TestNewAgentMonitor(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}
	agentMonitor, _ := NewAgentMonitor(conn, "autoscalingid", "launchconfid", "i-249b35ae")

	if agentMonitor == nil {
		t.Fatal("nil AgentMonitor")
	}

	if agentMonitor.agent.instanceId != "i-249b35ae" {
		t.Fatal("wrong AgentMonitor")
	}
}

func TestTerminateInstance(t *testing.T) {

	conn := &awsMock.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}
	agentMonitor, _ := NewAgentMonitor(conn, "autoscalingid", "launchconfid", "i-249b35ae")
	agentMonitor.Destroy()

	callArguments := conn.Requests["TerminateInstance"]

	if (*callArguments)[0] != "i-249b35ae" {
		t.Fatal("Incorrect parameters for TerminateInstance")
	}
}
