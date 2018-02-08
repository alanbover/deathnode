package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
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
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn, []string{"frameworkName1"})

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
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn, []string{"frameworkName1"})
		mesosMonitor.Refresh()

		constraint, _ := newConstraint("protectedConstraint")
		Convey("it should filter instances with protectedLabels or protectedFrameworks", func() {
			instances := constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instances), ShouldEqual, 1)
		})
	})
}

func TestFilterFrameworkContraint(t *testing.T) {

	Convey("When creating a filterFrameworkContraint", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node1", "node2", "node3"},
				"DescribeAGByName":     {"default"},
			},
		}
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn, []string{"frameworkName1", "frameworkName2"})
		mesosMonitor.Refresh()

		Convey("it should filter instances with tasks running those frameworks", func() {
			constraint, _ := newConstraint("filterFrameworkConstraint=frameworkName2")
			instances := constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instances), ShouldEqual, 2)

			constraint, _ = newConstraint("filterFrameworkConstraint=frameworkName1")
			instances = constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instances), ShouldEqual, 1)
		})
	})
}

func TestTaskNameRegexpConstraint(t *testing.T) {

	Convey("When creating a filterFrameworkContraint", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"node1", "node2", "node3"},
				"DescribeAGByName":     {"default"},
			},
		}
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		instanceMonitor, mesosMonitor := prepareMonitorsForConstraints(awsConn, mesosConn, []string{"frameworkName1", "frameworkName2"})
		mesosMonitor.Refresh()

		Convey("it should filter instances with tasks running those frameworks", func() {
			constraint, _ := newConstraint("taskNameRegexpConstraint=.*ask1")
			instances := constraint.filter(instanceMonitor.GetInstances(), mesosMonitor)
			So(len(instances), ShouldEqual, 2)
		})
	})
}

func prepareMonitorsForConstraints(
	awsConn *aws.ConnectionMock, mesosConn *mesos.ClientMock, protectedFrameworks []string) (
	*monitor.AutoscalingGroupMonitor, *monitor.MesosMonitor) {

	ctx := &context.ApplicationContext{
		AwsConn:   awsConn,
		MesosConn: mesosConn,
		Conf: context.ApplicationConf{
			DeathNodeMark:            "DEATH_NODE_MARK",
			AutoscalingGroupPrefixes: []string{"some-Autoscaling-Group"},
			ProtectedFrameworks:      protectedFrameworks,
		},
	}

	autoscalingGroups := monitor.NewAutoscalingServiceMonitor(ctx)
	autoscalingGroups.Refresh()
	return autoscalingGroups.GetAutoscalingGroupMonitorsList()[0], monitor.NewMesosMonitor(ctx)
}
