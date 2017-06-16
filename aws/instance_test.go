package aws

import (
	"testing"
)

func TestNewInstanceMonitor(t *testing.T) {

	conn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "i-249b35ae")

	if instanceMonitor == nil {
		t.Fatal("nil InstanceMonitor")
	}

	if instanceMonitor.instance.instanceId != "i-249b35ae" {
		t.Fatal("wrong InstanceMonitor")
	}

	if instanceMonitor.instance.markedToBeRemoved {
		t.Fatal("wrong markedToBeRemoved value")
	}
}

func TestTerminateInstance(t *testing.T) {

	conn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "i-249b35ae")
	instanceMonitor.Destroy()

	callArguments := conn.Requests["TerminateInstance"]

	if (callArguments)[0][0] != "i-249b35ae" {
		t.Fatal("Incorrect parameters for TerminateInstance")
	}
}

func TestSetInstanceTag(t *testing.T) {

	conn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "i-249b35ae")
	instanceMonitor.MarkToBeRemoved()

	callArguments := conn.Requests["SetInstanceTag"]

	if (callArguments)[0][0] != DEATH_NODE_TAG_MARK {
		t.Fatal("Incorrect tag key for SetTags")
	}

	if (callArguments)[0][2] != "i-249b35ae" {
		t.Fatal("Incorrect instance Id for SetTags")
	}
}

func TestInstanceMarkToBeRemoved(t *testing.T) {

	conn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"node_with_tag"},
		},
	}

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "i-249b35ae")

	if !instanceMonitor.instance.markedToBeRemoved {
		t.Fatal("wrong markedToBeRemoved value")
	}

}
