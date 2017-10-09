package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestDestroyInstanceAttempt(t *testing.T) {

	Convey("When running DestroyInstancesAttempt", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {
					"node1", "node2", "node3",
				},
				"DescribeInstancesByTag": {"default"},
				"DescribeAGByName":       {"default"},
			},
		}

		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		notebook := prepareRunParameters(awsConn, mesosConn, 0)

		Convey("if there is no instances marked to be removed", func() {
			Convey("it should do nothing", func() {
				notebook.DestroyInstancesAttempt()
				So(mesosConn.Requests["SetHostInMaintenance"], ShouldNotBeNil)
				So(awsConn.Requests["DetachInstance"], ShouldBeNil)
				So(awsConn.Requests["TerminateInstance"], ShouldBeNil)
			})
		})
		Convey("if there is a instance marked to be removed", func() {
			awsConn.Records = map[string]*[]string{
				"DescribeInstancesByTag": {"one_undesired_host"},
			}

			Convey("if it's still attached to an ASG, it should be dettached", func() {
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["DetachInstance"], ShouldNotBeNil)
			})
			Convey("if it has tasks running from protected frameworks, it shouldn't be removed", func() {
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["TerminateInstance"], ShouldBeNil)
			})
			Convey("if it has no task running from protected frameworks, it should be removed", func() {
				mesosConn.Records = map[string]*[]string{
					"GetMesosFrameworks": {"default"},
					"GetMesosSlaves":     {"default"},
					"GetMesosTasks":      {"notasks"},
				}
				notebook.mesosMonitor.Refresh()
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["TerminateInstance"], ShouldNotBeNil)
			})
		})
		Convey("if there is two instances marked to be removed", func() {
			awsConn.Records = map[string]*[]string{
				"DescribeInstancesByTag": {"two_undesired_hosts"},
			}
			Convey("both should be removed if no delayDeleteSeconds", func() {
				mesosConn.Records = map[string]*[]string{
					"GetMesosFrameworks": {"default"},
					"GetMesosSlaves":     {"default"},
					"GetMesosTasks":      {"notasks"},
				}
				notebook.mesosMonitor.Refresh()
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["TerminateInstance"], ShouldNotBeNil)
				So(len(awsConn.Requests["TerminateInstance"]), ShouldEqual, 2)
			})
			Convey("only one should be removed if delayDeleteSeconds", func() {
				notebook.delayDeleteSeconds = 100
				mesosConn.Records = map[string]*[]string{
					"GetMesosFrameworks": {"default"},
					"GetMesosSlaves":     {"default"},
					"GetMesosTasks":      {"notasks"},
				}
				notebook.mesosMonitor.Refresh()
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["TerminateInstance"], ShouldNotBeNil)
				So(len(awsConn.Requests["TerminateInstance"]), ShouldEqual, 1)
			})
		})
	})
}

func prepareRunParameters(awsConn aws.ClientInterface, mesosConn mesos.ClientInterface, delayDeleteSeconds int) *Notebook {

	protectedFrameworks := []string{"frameworkName1"}
	protectedTasksLabels := []string{"task1"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}
	mesosMonitor := monitor.NewMesosMonitor(mesosConn, protectedFrameworks, protectedTasksLabels)
	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, autoscalingGroupsNames, "DEATH_NODE_MARK")

	mesosMonitor.Refresh()
	autoscalingGroups.Refresh()

	notebook := NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds, "DEATH_NODE_MARK")
	return notebook
}
