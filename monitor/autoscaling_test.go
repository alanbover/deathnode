package monitor

import (
	"github.com/alanbover/deathnode/aws"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNewAutoscalingGroup(t *testing.T) {

	Convey("When creating a new autoscalingGroupMonitor", t, func() {
		monitor := newTestMonitor(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})

		Convey("it should not be nil", func() {
			So(monitor, ShouldNotBeNil)
		})
		Convey("it should have 3 instances", func() {
			So(len(monitor.autoscaling.instanceMonitors), ShouldEqual, 3)
		})
		Convey("it should have no undesired instances", func() {
			So(monitor.NumUndesiredInstances(), ShouldEqual, 0)
		})
		Convey("it should have a correct autoscalingGroup name", func() {
			So(monitor.autoscaling.autoscalingGroupName, ShouldEqual, "some-Autoscaling-Group")
		})
	})
}

func TestUndesiredInstances(t *testing.T) {

	Convey("When creating a new autoscalingGroupMonitor", t, func() {

		Convey("if it has 3 instances and desired instances are 3", func() {
			monitor := newTestMonitor(&aws.ConnectionMock{
				Records: map[string]*[]string{
					"DescribeInstanceById": {"default", "default", "default"},
					"DescribeAGByName":     {"default"},
				},
			})

			Convey("it should have no undesired instances", func() {
				So(monitor.NumUndesiredInstances(), ShouldEqual, 0)
			})
		})
		Convey("if it has 3 instances and desired instances are 2", func() {
			monitor := newTestMonitor(&aws.ConnectionMock{
				Records: map[string]*[]string{
					"DescribeInstanceById": {"default", "default", "default"},
					"DescribeAGByName":     {"one_undesired_host"},
				},
			})

			Convey("it should have one undesired instance", func() {
				So(monitor.NumUndesiredInstances(), ShouldEqual, 1)
			})
		})
	})
}

func TestRefresh(t *testing.T) {

	Convey("When refreshing an AutoscalingGroup", t, func() {
		Convey("and it changes", func() {
			monitors := newTestAutoscalingMonitors(&aws.ConnectionMock{
				Records: map[string]*[]string{
					"DescribeInstanceById": {
						"default", "default", "default", "default", "default", "default"},
					"DescribeAGByName": {"default", "refresh"},
				},
			})
			monitors.Refresh()
			monitor := monitors.GetAllMonitors()[0]
			Convey("it should have 3 instances", func() {
				So(len(monitor.autoscaling.instanceMonitors), ShouldEqual, 3)
			})
			Convey("it should have updated instance id's", func() {
				So(monitor.autoscaling.instanceMonitors, ShouldContainKey, "i-34719eb8")
				So(monitor.autoscaling.instanceMonitors, ShouldContainKey, "i-777a73cf")
				So(monitor.autoscaling.instanceMonitors, ShouldContainKey, "i-666ca923")
			})
			Convey("it should have one undesired instance", func() {
				So(monitor.NumUndesiredInstances(), ShouldEqual, 1)
			})
		})
		Convey("and a new one appears", func() {
			monitors := newTestAutoscalingMonitors(&aws.ConnectionMock{
				Records: map[string]*[]string{
					"DescribeInstanceById": {
						"default", "default", "default",
						"default", "default", "default",
						"default", "default", "default",
					},
					"DescribeAGByName": {"default", "two_asg", "default"},
				},
			})
			monitors.Refresh()
			Convey("two different autoscalingGroups should be monitored", func() {
				currentMonitors := monitors.GetAllMonitors()
				So(len(currentMonitors), ShouldEqual, 2)
				So(
					currentMonitors[0].autoscaling.autoscalingGroupName, ShouldNotEqual,
					currentMonitors[1].autoscaling.autoscalingGroupName)
			})
			Convey("it dissapears after a new refresh", func() {
				monitors.Refresh()
				So(len(monitors.GetAllMonitors()), ShouldEqual, 1)
			})
		})
	})
}

func TestSetInstanceProtection(t *testing.T) {

	Convey("When creating an AutoscalingGroup", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"instance_profile_disabled"},
			},
		}
		newTestAutoscalingMonitors(awsConn)
		callArguments := awsConn.Requests["SetASGInstanceProtection"]
		Convey("instances should have been set with instanceProtection flag", func() {
			So(callArguments, ShouldNotBeNil)
			So(len(callArguments), ShouldBeGreaterThanOrEqualTo, 1)
			So(len(callArguments[0]), ShouldBeGreaterThanOrEqualTo, 1)
			So(callArguments[0][0], ShouldEqual, "some-Autoscaling-Group")
			So(callArguments[0][1], ShouldEqual, "i-34719eb8")
		})
	})
}

func TestGetAutoscalingNameByInstanceId(t *testing.T) {

	Convey("GetAutoscalingNameByInstanceID should", t, func() {
		monitors := newTestAutoscalingMonitors(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})
		Convey("return an autoscalingGroupName if the instance belongs to a monitored Autoscaling", func() {
			asgName, found := monitors.GetAutoscalingNameByInstanceID("i-34719eb8")
			So(asgName, ShouldEqual, "some-Autoscaling-Group")
			So(found, ShouldBeTrue)
		})
		Convey("return not found if the instance doesn't belong to any monitored Autoscaling", func() {
			asgName, found := monitors.GetAutoscalingNameByInstanceID("i-doesntexist")
			So(asgName, ShouldEqual, "")
			So(found, ShouldBeFalse)
		})
	})
}

func TestGetInstances(t *testing.T) {

	Convey("When an autoscaling group with 3 instances is created", t, func() {
		monitor := newTestMonitor(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})
		Convey("GetInstances should return three instances", func() {
			So(len(monitor.GetInstances()), ShouldEqual, 3)
		})
		Convey("but after mark on instance to be deleted", func() {
			instanceToBeMarked := monitor.GetInstances()[0]
			instanceToBeMarked.MarkToBeRemoved()
			Convey("GetInstances should not return it", func() {
				So(instanceToBeMarked, ShouldNotBeIn, monitor.GetInstances())
			})
		})
		Convey("but after delete one instance", func() {
			instanceToBeMarked := monitor.GetInstances()[0]
			monitor.RemoveInstance(instanceToBeMarked)
			Convey("GetInstances should not return it", func() {
				So(instanceToBeMarked, ShouldNotBeIn, monitor.GetInstances())
			})
		})
	})
}

func newTestMonitor(awsConn *aws.ConnectionMock) *AutoscalingGroupMonitor {

	return newTestAutoscalingMonitors(awsConn).GetAllMonitors()[0]
}

func newTestAutoscalingMonitors(awsConn *aws.ConnectionMock) *AutoscalingGroupsMonitor {

	autoscalingGroupNames := []string{"some-Autoscaling-Group"}
	autoscalingGroups, _ := NewAutoscalingGroupMonitors(awsConn, autoscalingGroupNames, "DEATH_NODE_MARK")
	autoscalingGroups.Refresh()
	return autoscalingGroups
}
