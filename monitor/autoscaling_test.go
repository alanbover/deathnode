package monitor

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
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
			So(len(monitor.instanceMonitors), ShouldEqual, 3)
		})
		Convey("it should have no undesired instances", func() {
			So(monitor.GetNumUndesiredInstances(), ShouldEqual, 0)
		})
		Convey("it should have a correct autoscalingGroup name", func() {
			So(monitor.autoscalingGroupName, ShouldEqual, "some-Autoscaling-Group")
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
				So(monitor.GetNumUndesiredInstances(), ShouldEqual, 0)
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
				So(monitor.GetNumUndesiredInstances(), ShouldEqual, 1)
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
			monitor := monitors.GetAutoscalingGroupMonitorsList()[0]
			Convey("it should have 3 instances", func() {
				So(len(monitor.instanceMonitors), ShouldEqual, 3)
			})
			Convey("it should have updated instance id's", func() {
				So(monitor.instanceMonitors, ShouldContainKey, "i-34719eb8")
				So(monitor.instanceMonitors, ShouldContainKey, "i-777a73cf")
				So(monitor.instanceMonitors, ShouldContainKey, "i-666ca923")
			})
			Convey("it should have no undesired instances", func() {
				So(monitor.GetNumUndesiredInstances(), ShouldEqual, 0)
			})
			Convey("lifecycleState should have been updated", func() {
				lcState := monitor.instanceMonitors["i-34719eb8"].lifecycleState
				So(lcState, ShouldEqual, LifecycleStateTerminatingWait)
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
				currentMonitors := monitors.GetAutoscalingGroupMonitorsList()
				So(len(currentMonitors), ShouldEqual, 2)
				So(
					currentMonitors[0].autoscalingGroupName, ShouldNotEqual,
					currentMonitors[1].autoscalingGroupName)
			})
			Convey("it dissapears after a new refresh", func() {
				monitors.Refresh()
				So(len(monitors.GetAutoscalingGroupMonitorsList()), ShouldEqual, 1)
			})
		})
	})
}

func TestInitializeAutoscalingGroup(t *testing.T) {

	Convey("When creating an AutoscalingGroup", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"instance_profile_disabled"},
				"HasLifeCycleHook":     {"false"},
			},
		}
		newTestAutoscalingMonitors(awsConn)
		Convey("instances should have been set with instanceProtection flag", func() {
			callArguments := awsConn.Requests["SetASGInstanceProtection"]
			So(callArguments, ShouldNotBeNil)
			So(len(callArguments), ShouldBeGreaterThanOrEqualTo, 1)
			So(len(callArguments[0]), ShouldBeGreaterThanOrEqualTo, 1)
			So(callArguments[0][0], ShouldEqual, "some-Autoscaling-Group")
			So(callArguments[0][1], ShouldEqual, "i-34719eb8")
		})
		Convey("a lifecycleHook should have been added", func() {
			callArguments := awsConn.Requests["PutLifeCycleHook"]
			So(len(callArguments), ShouldEqual, 1)
			So(len(callArguments[0]), ShouldBeGreaterThanOrEqualTo, 1)
			So(callArguments[0][0], ShouldEqual, "some-Autoscaling-Group")
			So(callArguments[0][1], ShouldEqual, "3600")
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
			instanceToBeMarked.TagToBeRemoved()
			Convey("GetInstances should not return it", func() {
				So(instanceToBeMarked, ShouldNotBeIn, monitor.GetInstances())
			})
		})
		Convey("but after delete one instance", func() {
			instanceToBeMarked := monitor.GetInstances()[0]
			delete(monitor.instanceMonitors, instanceToBeMarked.instanceID)
			Convey("GetInstances should not return it", func() {
				So(instanceToBeMarked, ShouldNotBeIn, monitor.GetInstances())
			})
		})
	})
}

func TestGetInstanceById(t *testing.T) {

	Convey("When an autoscaling group with 3 instances is created", t, func() {

		monitors := newTestAutoscalingMonitors(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})
		Convey("GetInstanceById should return an instanceMonitor if it exists", func() {
			instance, _ := monitors.GetInstanceByID("i-34719eb8")
			So(*instance.InstanceID(), ShouldEqual, "i-34719eb8")
		})
		Convey("GetInstanceById should return error if instance doesn't exist", func() {
			_, err := monitors.GetInstanceByID("i-doesntexist")
			So(err, ShouldNotBeNil)
		})
	})
}

func newTestMonitor(awsConn *aws.ConnectionMock) *AutoscalingGroupMonitor {

	return newTestAutoscalingMonitors(awsConn).GetAutoscalingGroupMonitorsList()[0]
}

func newTestAutoscalingMonitors(awsConn *aws.ConnectionMock) *AutoscalingServiceMonitor {

	ctx := &context.ApplicationContext{
		AwsConn: awsConn,
		Conf: context.ApplicationConf{
			DeathNodeMark:            "DEATH_NODE_MARK",
			AutoscalingGroupPrefixes: []string{"some-Autoscaling-Group"},
		},
	}

	autoscalingGroups := NewAutoscalingServiceMonitor(ctx)
	autoscalingGroups.Refresh()
	return autoscalingGroups
}
