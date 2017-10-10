package monitor

import (
	"testing"
	"github.com/alanbover/deathnode/aws"
	. "github.com/smartystreets/goconvey/convey"
)

func TestNewInstanceMonitor(t *testing.T) {

	Convey("When creating a new instanceMonitor", t, func() {
		conn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default"},
			},
		}
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK", "InService", false)

		Convey("it should not be nil", func() {
			So(monitor, ShouldNotBeNil)
		})
		Convey("it should have a correct instanceId", func() {
			So(monitor.instance.instanceID, ShouldEqual, "i-249b35ae")
		})
		Convey("it shouldn't be marked to be removed", func() {
			So(monitor.instance.isMarkedToBeRemoved, ShouldBeFalse)
		})
		Convey("and MarkToBeRemoved is called", func() {
			monitor.MarkToBeRemoved()
			Convey("SetInstanceTag should be called with correct parameters", func() {
				callArguments := conn.Requests["SetInstanceTag"]
				So(callArguments[0][0], ShouldEqual, "DEATH_NODE_MARK")
				So(callArguments[0][2], ShouldEqual, "i-249b35ae")
			})
		})
		Convey("GetIP should return it's ip", func() {
			So(monitor.GetIP(), ShouldEqual, "10.0.0.2")
		})
		Convey("GetInstanceID should return it's ip", func() {
			So(*monitor.GetInstanceID(), ShouldEqual, "i-249b35ae")
		})
	})
}

func TestInstanceMarkToBeRemoved(t *testing.T) {

	Convey("When creating a new instanceMonitor that is marked to be removed", t, func() {
		conn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node_with_tag"},
			},
		}
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK", "InService", false)
		Convey("and isMarkToBeRemoved is called", func() {
			So(monitor.instance.isMarkedToBeRemoved, ShouldBeTrue)
		})
	})
}

func TestInstanceProtection(t *testing.T) {

	Convey("When creating a new instanceMonitor with instance protection true", t, func() {
		conn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node_with_tag"},
			},
		}
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK", "InService", true)
		Convey("instance should have instanceProtection", func() {
			So(monitor.instance.isProtected, ShouldBeTrue)
		})
		Convey("and RemoveInstanceProtection is called", func() {
			monitor.RemoveInstanceProtection()
			Convey("instance should not have instanceProtection", func() {
				So(monitor.instance.isProtected, ShouldBeFalse)
			})
			Convey("RemoveASGInstanceProtection aws should have been called", func() {
				So(conn.Requests, ShouldContainKey, "RemoveASGInstanceProtection")
				So(len(conn.Requests["RemoveASGInstanceProtection"]), ShouldEqual, 1)
				So(conn.Requests["RemoveASGInstanceProtection"][0][1], ShouldEqual, "i-249b35ae")
				So(conn.Requests["RemoveASGInstanceProtection"][0][0], ShouldEqual, "autoscalingid")
			})
		})
	})
}

func TestLifecycleState(t *testing.T) {

	Convey("When creating a new instanceMonitor", t, func() {
		conn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default"},
			},
		}
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK", "InService", false)
		Convey("and we call SetLifecycleState", func() {
			monitor.setLifecycleState("Terminating:Wait")
			Convey("instance should have net LifecycleState value", func() {
				So(monitor.instance.lifecycleState, ShouldEqual, "Terminating:Wait")
			})
		})
	})
}
