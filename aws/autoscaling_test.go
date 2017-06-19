package aws

import (
	"testing"
)

func TestNewAutoscalingGroupMonitor(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default", "default", "default"},
			"DescribeAGByName":     {"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()
	monitor := autoscalingGroups.GetMonitors()[0]

	if monitor == nil {
		t.Fatal("TestNewAutoscalingGroupMonitor return nil")
	}

	if (len(monitor.autoscaling.instanceMonitors)) != 3 {
		t.Fatal("Incorrect number of instances in ASG. Found: ", len(monitor.autoscaling.instanceMonitors))
	}

	if monitor.NumUndesiredInstances() > 0 {
		t.Fatal("Incorrect number of undesired ASG instances. Found: ", monitor.NumUndesiredInstances())
	}

	if awsConn.Requests["SetASGInstanceProtection"] != nil {
		t.Fatal("Method SetASGInstanceProtection should have not been called")
	}
}

func TestNumUndesiredASGinstances(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default", "default", "default"},
			"DescribeAGByName":     {"one_undesired_host"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()
	monitor := autoscalingGroups.GetMonitors()[0]

	if monitor.NumUndesiredInstances() != 1 {
		t.Fatal("Incorrect number of undesired ASG instances. Found: ", monitor.NumUndesiredInstances())
	}
}

func TestNewAutoscalingGroups(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default", "default", "default"},
			"DescribeAGByName":     {"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	if (autoscalingGroups.GetMonitors())[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" {
		t.Fatal("Error creating AutoscalingGroups")
	}
}

func TestRefresh(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default", "default", "default", "default", "default", "default"},
			"DescribeAGByName":     {"default", "refresh"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	autoscalingGroup := (autoscalingGroups.GetMonitors())[0]
	autoscalingGroups.Refresh()

	if len(autoscalingGroup.autoscaling.instanceMonitors) != 3 {
		t.Fatal("Incorrect number of elements after refresh()")
	}

	expectedInstanceIDs := []string{"i-34719eb8", "i-777a73cf", "i-666ca923"}

	for _, instanceID := range expectedInstanceIDs {
		_, ok := autoscalingGroup.autoscaling.instanceMonitors[instanceID]
		if !ok {
			t.Fatal("Incorrect instanceId found after refresh()")
		}
	}

	if autoscalingGroup.NumUndesiredInstances() != 1 {
		t.Fatal("After refresh we should have one undesired instance")
	}
}

func TestSetInstanceProtection(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"default", "default", "default"},
			"DescribeAGByName":     {"instance_profile_disabled"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	callArguments := awsConn.Requests["SetASGInstanceProtection"]

	if callArguments == nil {
		t.Fatal("Method SetASGInstanceProtection should have been called")
	}

	if len(callArguments) < 1 {
		t.Fatal("Method SetASGInstanceProtection called with not enought arguments")
	}

	if callArguments[0][0] != "some-Autoscaling-Group" {
		t.Fatal("Method SetASGInstanceProtection called with incorrect autoscaling group")
	}

	if callArguments[0][1] != "i-34719eb8" {
		t.Fatal("Method SetASGInstanceProtection called with incorrect instanceId")
	}
}

func TestTwoAutoscalingMonitors(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"default", "default", "default",
				"default", "default", "default",
			},
			"DescribeAGByName": {"default", "two_asg"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()
	autoscalingGroups.Refresh()
	monitors := autoscalingGroups.GetMonitors()

	if len(monitors) != 2 {
		t.Fatal("Incorrect number autoscalingGroups")
	}

	if monitors[0].autoscaling.autoscalingGroupName == monitors[1].autoscaling.autoscalingGroupName {
		t.Fatal("Incorrect autoscaling group name")
	}

	if monitors[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" &&
		monitors[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group2" {
		t.Fatal("Incorrect autoscaling group name")
	}

	if monitors[1].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" &&
		monitors[1].autoscaling.autoscalingGroupName != "some-Autoscaling-Group2" {
		t.Fatal("Incorrect autoscaling group name")
	}
}

func TestRemoveAutoscalingGroup(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"default", "default", "default",
				"default", "default", "default",
				"default", "default", "default",
			},
			"DescribeAGByName": {"default", "two_asg", "default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()
	autoscalingGroups.Refresh()
	autoscalingGroups.Refresh()
	monitors := autoscalingGroups.GetMonitors()

	if len(monitors) != 1 {
		t.Fatal("Incorrect number autoscalingGroups")
	}
}

func TestGetAutoscalingNameByInstanceId(t *testing.T) {

	awsConn := &ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {"node_with_tag", "node_with_tag", "node_with_tag"},
			"DescribeAGByName":     {"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	asgID, _ := autoscalingGroups.GetAutoscalingNameByInstanceID("i-34719eb8")

	if asgID != "some-Autoscaling-Group" {
		t.Errorf("Expected : [%s] Found: [%s]", "some-Autoscaling-Group", asgID)
	}

	_, found := autoscalingGroups.GetAutoscalingNameByInstanceID("i-doesntexist")

	if found {
		t.Errorf("Expected : [%b] Found: [%s]", found, "false")
	}

}
