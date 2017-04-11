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

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "launchconfid", "i-249b35ae")

	if instanceMonitor == nil {
		t.Fatal("nil InstanceMonitor")
	}

	if instanceMonitor.instance.instanceId != "i-249b35ae" {
		t.Fatal("wrong InstanceMonitor")
	}
}

func TestTerminateInstance(t *testing.T) {

	conn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default"},
		},
	}

	instanceMonitor, _ := NewInstanceMonitor(conn, "autoscalingid", "launchconfid", "i-249b35ae")
	instanceMonitor.Destroy()

	callArguments := conn.Requests["TerminateInstance"]

	if (*callArguments)[0] != "i-249b35ae" {
		t.Fatal("Incorrect parameters for TerminateInstance")
	}
}
