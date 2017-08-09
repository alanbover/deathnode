package aws

import (
	"testing"
)

func TestNewInstanceMonitor(t *testing.T) {

	conn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default"},
		},
	}

	instanceMonitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK")

	if instanceMonitor == nil {
		t.Fatal("nil InstanceMonitor")
	}

	if instanceMonitor.instance.instanceID != "i-249b35ae" {
		t.Fatal("wrong InstanceMonitor")
	}

	if instanceMonitor.instance.markedToBeRemoved {
		t.Fatal("wrong markedToBeRemoved value")
	}
}

func TestSetInstanceTag(t *testing.T) {

	conn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default"},
		},
	}

	instanceMonitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK")
	instanceMonitor.MarkToBeRemoved()

	callArguments := conn.Requests["SetInstanceTag"]

	if (callArguments)[0][0] != "DEATH_NODE_MARK" {
		t.Fatal("Incorrect tag key for SetTags")
	}

	if (callArguments)[0][2] != "i-249b35ae" {
		t.Fatal("Incorrect instance Id for SetTags")
	}
}

func TestInstanceMarkToBeRemoved(t *testing.T) {

	conn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"node_with_tag"},
		},
	}

	instanceMonitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK")

	if !instanceMonitor.instance.markedToBeRemoved {
		t.Fatal("wrong markedToBeRemoved value")
	}

}
