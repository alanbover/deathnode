package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"testing"
	"github.com/alanbover/deathnode/monitor"
	"github.com/alanbover/deathnode/mesos"
	. "github.com/smartystreets/goconvey/convey"
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
		notebook := newNotebook(awsConn, mesosConn, 0)

		Convey("if there is no instances marked to be removed", func() {
			Convey("it should do nothing", func() {
				notebook.DestroyInstancesAttempt()
				So(mesosConn.Requests["SetHostInMaintenance"], ShouldNotBeNil)
				So(awsConn.Requests["DetachInstance"], ShouldBeNil)
				So(awsConn.Requests["TerminateInstance"], ShouldBeNil)
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldBeNil)
				So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldBeNil)
			})
		})
		Convey("if there is a instance marked to be removed", func() {
			awsConn.Records = map[string]*[]string{
				"DescribeInstancesByTag":       {"one_undesired_host", "one_undesired_host"},
			}
			instanceMonitor, _ := notebook.autoscalingGroups.GetInstanceByID("i-34719eb8")
			Convey("Check remove instance protection flags", func() {
				notebook.DestroyInstancesAttempt()
				Convey("if it has instanceProtection flag, it should be removed", func() {
					So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldNotBeNil)
					So(instanceMonitor.IsProtected(), ShouldBeFalse)
				})
				Convey("if it doesn't have instanceProtection flag, it should do nothing", func() {
					notebook.DestroyInstancesAttempt()
					So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldNotBeNil)
					So(len(awsConn.Requests["RemoveASGInstanceProtection"]), ShouldEqual, 1)
					So(instanceMonitor.IsProtected(), ShouldBeFalse)
				})
			})
			Convey("Check completeLifeCycle", func() {
				Convey("if it has tasks running from protected frameworks, completeLifeCycle should not be called", func() {
					notebook.DestroyInstancesAttempt()
					So(awsConn.Requests["CompleteLifecycleAction"], ShouldBeNil)
				})
				Convey("if it has no task running from protected frameworks, ", func() {
					mesosConn.Records = map[string]*[]string{
						"GetMesosFrameworks": {"default"},
						"GetMesosSlaves":     {"default"},
						"GetMesosTasks":      {"notasks"},
					}
					notebook.mesosMonitor.Refresh()
					notebook.DestroyInstancesAttempt()
					Convey("completeLifeCycle should not be called if instance lifeCycleState is not in waiting state", func() {
						So(awsConn.Requests["CompleteLifecycleAction"], ShouldBeNil)
					})
					Convey("completeLifeCycle should be called only after instance lifeCycleState is in waiting state", func() {
						awsConn.Records = map[string]*[]string{
							"DescribeInstanceById": {
								"node1", "node2", "node3",
							},
							"DescribeInstancesByTag": {"one_undesired_host"},
							"DescribeAGByName":       {"one_undesired_host_one_terminating"},
						}
						notebook.autoscalingGroups.Refresh()
						notebook.DestroyInstancesAttempt()
						So(awsConn.Requests["CompleteLifecycleAction"], ShouldNotBeNil)
					})
				})
			})
		})
		Convey("if there is two instances marked to be removed", func() {
			awsConn.Records = map[string]*[]string{
				"DescribeInstanceById": {
					"node1", "node2", "node3",
				},
				"DescribeInstancesByTag": {"two_undesired_hosts"},
				"DescribeAGByName":       {"two_undesired_hosts_two_terminating"},
			}
			notebook.autoscalingGroups.Refresh()
			Convey("both should be removed if no delayDeleteSeconds", func() {
				mesosConn.Records = map[string]*[]string{
					"GetMesosFrameworks": {"default"},
					"GetMesosSlaves":     {"default"},
					"GetMesosTasks":      {"notasks"},
				}
				notebook.mesosMonitor.Refresh()
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldNotBeNil)
				So(len(awsConn.Requests["CompleteLifecycleAction"]), ShouldEqual, 2)
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
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldNotBeNil)
				So(len(awsConn.Requests["CompleteLifecycleAction"]), ShouldEqual, 1)
			})
		})
	})
}

func newNotebook(awsConn aws.ClientInterface, mesosConn mesos.ClientInterface, delayDeleteSeconds int) *Notebook {

	protectedFrameworks := []string{"frameworkName1"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}
	mesosMonitor := monitor.NewMesosMonitor(mesosConn, protectedFrameworks)
	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, autoscalingGroupsNames, "DEATH_NODE_MARK")

	mesosMonitor.Refresh()
	autoscalingGroups.Refresh()

	notebook := NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds, "DEATH_NODE_MARK")
	return notebook
}
