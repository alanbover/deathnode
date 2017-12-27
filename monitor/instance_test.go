package monitor

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
	"github.com/benbjohnson/clock"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNewInstanceMonitor(t *testing.T) {

	Convey("When creating a new instanceMonitor", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default"},
			},
		}
		ctx := &context.ApplicationContext{
			AwsConn: awsConn,
			Conf: context.ApplicationConf{
				DeathNodeMark: "DEATH_NODE_MARK",
			},
			Clock: clock.New(),
		}

		monitor, _ := newInstanceMonitor(ctx, "autoscalingid", "i-249b35ae", "InService", false)

		Convey("it should not be nil", func() {
			So(monitor, ShouldNotBeNil)
		})
		Convey("it should have a correct instanceId", func() {
			So(monitor.instanceID, ShouldEqual, "i-249b35ae")
		})
		Convey("it shouldn't be marked to be removed", func() {
			So(monitor.IsMarkedToBeRemoved(), ShouldBeFalse)
		})
		Convey("and MarkToBeRemoved is called", func() {
			monitor.TagToBeRemoved()
			Convey("SetInstanceTag should be called with correct parameters", func() {
				callArguments := awsConn.Requests["SetInstanceTag"]
				So(callArguments[0][0], ShouldEqual, "DEATH_NODE_MARK")
				So(callArguments[0][2], ShouldEqual, "i-249b35ae")
			})
		})
		Convey("GetIP should return it's ip", func() {
			So(monitor.IP(), ShouldEqual, "10.0.0.2")
		})
		Convey("GetInstanceID should return it's ip", func() {
			So(*monitor.InstanceID(), ShouldEqual, "i-249b35ae")
		})
	})
}

func TestInstanceMarkToBeRemoved(t *testing.T) {

	Convey("When creating a new instanceMonitor that is marked to be removed", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node_with_tag"},
			},
		}

		ctx := &context.ApplicationContext{
			AwsConn: awsConn,
			Conf: context.ApplicationConf{
				DeathNodeMark: "DEATH_NODE_MARK",
			},
			Clock: clock.New(),
		}

		monitor, _ := newInstanceMonitor(ctx, "autoscalingid", "i-249b35ae", "InService", false)
		Convey("and isMarkToBeRemoved is called", func() {
			So(monitor.IsMarkedToBeRemoved(), ShouldBeTrue)
		})
	})
}

func TestInstanceProtection(t *testing.T) {

	Convey("When creating a new instanceMonitor with instance protection true", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node_with_tag"},
			},
		}

		ctx := &context.ApplicationContext{
			AwsConn: awsConn,
			Conf: context.ApplicationConf{
				DeathNodeMark: "DEATH_NODE_MARK",
			},
			Clock: clock.New(),
		}

		monitor, _ := newInstanceMonitor(ctx, "autoscalingid", "i-249b35ae", "InService", true)
		Convey("instance should have instanceProtection", func() {
			So(monitor.isProtected, ShouldBeTrue)
		})
		Convey("and RemoveInstanceProtection is called", func() {
			monitor.RemoveInstanceProtection()
			Convey("instance should not have instanceProtection", func() {
				So(monitor.isProtected, ShouldBeFalse)
			})
			Convey("RemoveASGInstanceProtection aws should have been called", func() {
				So(awsConn.Requests, ShouldContainKey, "RemoveASGInstanceProtection")
				So(len(awsConn.Requests["RemoveASGInstanceProtection"]), ShouldEqual, 1)
				So(awsConn.Requests["RemoveASGInstanceProtection"][0][1], ShouldEqual, "i-249b35ae")
				So(awsConn.Requests["RemoveASGInstanceProtection"][0][0], ShouldEqual, "autoscalingid")
			})
		})
	})
}

func TestLifecycleState(t *testing.T) {

	Convey("When creating a new instanceMonitor", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default"},
			},
		}

		ctx := &context.ApplicationContext{
			AwsConn: awsConn,
			Conf: context.ApplicationConf{
				DeathNodeMark: "DEATH_NODE_MARK",
			},
			Clock: clock.New(),
		}

		monitor, _ := newInstanceMonitor(ctx, "autoscalingid", "i-249b35ae", "InService", true)
		Convey("and we call SetLifecycleState", func() {
			Convey("when the instance has instanceProtection enabled", func() {
				monitor.setLifecycleState(LifecycleStateTerminatingWait)
				Convey("instance should have new LifecycleState value", func() {
					So(monitor.lifecycleState, ShouldEqual, LifecycleStateTerminatingWait)
				})
				Convey("MarkToBeRemoved should have been called", func() {
					So(awsConn.Requests["SetInstanceTag"], ShouldNotBeNil)
				})
			})
			Convey("when the instance has instanceProtection disabled", func() {
				monitor.isProtected = false
				monitor.setLifecycleState(LifecycleStateTerminatingWait)
				Convey("instance should have new LifecycleState value", func() {
					So(monitor.lifecycleState, ShouldEqual, LifecycleStateTerminatingWait)
				})
				Convey("MarkToBeRemoved should not have been called", func() {
					So(awsConn.Requests["SetInstanceTag"], ShouldBeNil)
				})
			})
		})
	})
}
