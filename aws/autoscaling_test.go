package aws

import (
	"testing"
)

func TestNewAutoscalingGroupMonitor(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
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

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"one_undesired_host"},
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

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	if (autoscalingGroups.GetMonitors())[0].autoscaling.autoscalingGroupName != "some-Autoscaling-Group" {
		t.Fatal("Error creating AutoscalingGroups")
	}
}

func TestRemoveInstanceFromAutoscalingGroup(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	instanceMonitors := (autoscalingGroups.GetMonitors())[0].autoscaling.instanceMonitors
	instanceMonitors["i-34719eb8"].RemoveFromAutoscalingGroup()

	callArguments := awsConn.Requests["DetachInstance"]

	if (callArguments)[0][0] != "some-Autoscaling-Group" || (callArguments)[0][1] != "i-34719eb8" {
		t.Fatal("Incorrect parameters when calling RemoveFromAutoscalingGroup")
	}
}

func TestRefresh(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default", "default", "default", "default"},
			"DescribeAGByName":     &[]string{"default", "refresh"},
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

	expectedInstanceIds := []string{"i-34719eb8", "i-777a73cf", "i-666ca923"}

	for _, instanceId := range expectedInstanceIds {
		_, ok := autoscalingGroup.autoscaling.instanceMonitors[instanceId]
		if !ok {
			t.Fatal("Incorrect instanceId found after refresh()")
		}
	}

	if autoscalingGroup.NumUndesiredInstances() != 1 {
		t.Fatal("After refresh we should have one undesired instance")
	}
}

func TestSetInstanceProtection(t *testing.T) {

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"default", "default", "default"},
			"DescribeAGByName":     &[]string{"instance_profile_disabled"},
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

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"default", "default", "default",
				"default", "default", "default",
			},
			"DescribeAGByName": &[]string{"default", "two_asg"},
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

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"default", "default", "default",
				"default", "default", "default",
				"default", "default", "default",
			},
			"DescribeAGByName": &[]string{"default", "two_asg", "default"},
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

	awsConn := &AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{"node_with_tag", "node_with_tag", "node_with_tag"},
			"DescribeAGByName":     &[]string{"default"},
		},
	}

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroups(awsConn, autoscalingGroupNames)
	autoscalingGroups.Refresh()

	asgId, _ := autoscalingGroups.GetAutoscalingNameByInstanceId("i-34719eb8")

	if asgId != "some-Autoscaling-Group" {
		t.Errorf("Expected : [%s] Found: [%s]", "some-Autoscaling-Group", asgId)
	}

	_, found := autoscalingGroups.GetAutoscalingNameByInstanceId("i-doesntexist")

	if found {
		t.Errorf("Expected : [%b] Found: [%s]", "false", found)
	}

}
