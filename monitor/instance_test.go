package monitor

import (
	"github.com/alanbover/deathnode/aws"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestNewInstanceMonitor(t *testing.T) {

	Convey("When creating a new instanceMonitor", t, func() {
		conn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default"},
			},
		}
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK")

		Convey("it should not be nil", func() {
			So(monitor, ShouldNotBeNil)
		})
		Convey("it should have a correct instanceId", func() {
			So(monitor.instance.instanceID, ShouldEqual, "i-249b35ae")
		})
		Convey("it shouldn't be marked to be removed", func() {
			So(monitor.instance.markedToBeRemoved, ShouldBeFalse)
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
			So(monitor.GetInstanceID(), ShouldEqual, "i-249b35ae")
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
		monitor, _ := newInstanceMonitor(conn, "autoscalingid", "i-249b35ae", "DEATH_NODE_MARK")
		Convey("and MarkToBeRemoved is called", func() {
			So(monitor.instance.markedToBeRemoved, ShouldBeTrue)
		})
	})
}
