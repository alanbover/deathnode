package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestConstraints(t *testing.T) {

	Convey("When creating a constraint", t, func() {

		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		}
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{},
		}
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn)

		Convey("it should raise an issue if the constrant doesn't exist", func() {
			_, err := newConstraint("noExistingConstraint")
			So(err, ShouldNotBeNil)
		})
		Convey("if it's a noConstraintType, it just return all it's instances", func() {
			constraint, _ := newConstraint("noContraint")
			instances := constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instanceMonitor.GetInstances()), ShouldEqual, len(instances))
		})
	})
}

func TestProtectedConstraint(t *testing.T) {

	Convey("When creating a protectedConstraint", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node1", "node2", "node3"},
				"DescribeAGByName":     {"default"},
			},
		}
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": { "default" },
				"GetMesosSlaves":     { "default" },
				"GetMesosTasks":      { "default" },
			},
		}
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn)
		mesosMonitor.Refresh()

		constraint, _ := newConstraint("protectedConstraint")
		Convey("it should filter instances with protectedLabels out protectedFrameworks", func() {
			instances := constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instances), ShouldEqual, 1)
		})
	})
}

func prepareMonitorsForConstraints(awsConn *aws.ConnectionMock, mesosConn *mesos.ClientMock) (*monitor.AutoscalingGroupMonitor, *monitor.MesosMonitor) {

	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, []string{"some-Autoscaling-Group"}, "DEATH_NODE_MARK")
	autoscalingGroups.Refresh()

	mesosMonitor := monitor.NewMesosMonitor(mesosConn, []string{"frameworkName1"}, []string{"DEATHNODE_PROTECTED"})
	return autoscalingGroups.GetAllMonitors()[0], mesosMonitor
}
