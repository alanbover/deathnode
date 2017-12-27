package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	"github.com/benbjohnson/clock"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
	"time"
)

func TestRecordLifecycleActionHeartbeat(t *testing.T) {

	clockMock := clock.NewMock()
	clockMock.Set(time.Unix(1190995200, 0))

	Convey("When running DestroyInstancesAttempt", t, func() {
		awsConn := &aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {
					"node_with_tag", "node2", "node3",
				},
				"DescribeInstancesByTag": {"one_undesired_host"},
				"DescribeAGByName":       {"one_undesired_host_one_terminating"},
			},
		}

		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		notebook := newNotebook(awsConn, mesosConn, 0, clockMock)
		Convey("it should do nothing if no lifecycle to be refreshed", func() {
			clockMock.Set(time.Unix(1190997840, 0))
			notebook.DestroyInstancesAttempt()
			clockMock.Set(time.Unix(1190995200, 0))
			So(awsConn.Requests["RecordLifecycleActionHeartbeat"], ShouldBeNil)
		})
		Convey("it should refresh lifecycle if time is close to be expired", func() {
			clockMock.Set(time.Unix(1190997960, 0))
			notebook.DestroyInstancesAttempt()
			clockMock.Set(time.Unix(1190995200, 0))
			So(awsConn.Requests["RecordLifecycleActionHeartbeat"], ShouldNotBeNil)
		})
	})
}

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
		notebook := newNotebook(awsConn, mesosConn, 0, clock.New())

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
				"DescribeInstancesByTag": {"one_undesired_host", "one_undesired_host"},
			}
			instanceMonitor, _ := notebook.autoscalingGroups.GetInstanceByID("i-34719eb8")
			Convey("Check remove instance protection flags", func() {
				notebook.DestroyInstancesAttempt()
				Convey("if it has instanceProtection flag, it should be removed", func() {
					So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldNotBeNil)
					So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldHaveLength, 1)
					So(instanceMonitor.IsProtected(), ShouldBeFalse)
				})
				Convey("if it doesn't have instanceProtection flag", func() {
					Convey("it should not remove more instanceProtection flag", func() {
						notebook.DestroyInstancesAttempt()
						So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldNotBeNil)
						So(awsConn.Requests["RemoveASGInstanceProtection"], ShouldHaveLength, 1)
						So(instanceMonitor.IsProtected(), ShouldBeFalse)
					})
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
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldHaveLength, 2)
			})
			Convey("only one should be removed if delayDeleteSeconds", func() {
				notebook.ctx.Conf.DelayDeleteSeconds = 100
				mesosConn.Records = map[string]*[]string{
					"GetMesosFrameworks": {"default"},
					"GetMesosSlaves":     {"default"},
					"GetMesosTasks":      {"notasks"},
				}
				notebook.mesosMonitor.Refresh()
				notebook.DestroyInstancesAttempt()
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldNotBeNil)
				So(awsConn.Requests["CompleteLifecycleAction"], ShouldHaveLength, 1)
			})
		})
	})
}

func newNotebook(awsConn aws.ClientInterface, mesosConn mesos.ClientInterface, delayDeleteSeconds int, clk clock.Clock) *Notebook {

	ctx := &context.ApplicationContext{
		Clock:     clk,
		AwsConn:   awsConn,
		MesosConn: mesosConn,
		Conf: context.ApplicationConf{
			DeathNodeMark:            "DEATH_NODE_MARK",
			AutoscalingGroupPrefixes: []string{"some-Autoscaling-Group"},
			ProtectedFrameworks:      []string{"frameworkName1"},
			ProtectedTasksLabels:     []string{"task1"},
			DelayDeleteSeconds:       delayDeleteSeconds,
		},
	}

	mesosMonitor := monitor.NewMesosMonitor(ctx)
	mesosMonitor.Refresh()

	autoscalingGroups := monitor.NewAutoscalingServiceMonitor(ctx)
	autoscalingGroups.Refresh()

	notebook := NewNotebook(ctx, autoscalingGroups, mesosMonitor)
	return notebook
}
