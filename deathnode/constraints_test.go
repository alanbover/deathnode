package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/monitor"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestConstraints(t *testing.T) {

	Convey("When creating a constraint", t, func() {

		monitor := newTestMonitor(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})
		Convey("it should raise an issue if the constrant doesn't exist", func() {
			_, err := newConstraint("noExistingConstraint")
			So(err, ShouldNotBeNil)
		})
		Convey("if it's a noConstraintType, it just return all it's instances", func() {
			constraint, _ := newConstraint("noContraint")
			instances := constraint.filter(monitor.GetInstances())
			So(len(monitor.GetInstances()), ShouldEqual, len(instances))
		})
	})
}

func newTestMonitor(awsConn *aws.ConnectionMock) *monitor.AutoscalingGroupMonitor {

	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, []string{"some-Autoscaling-Group"}, "DEATH_NODE_MARK")
	autoscalingGroups.Refresh()
	return autoscalingGroups.GetAllMonitors()[0]
}
