package deathnode

import (
	"github.com/alanbover/deathnode/aws"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/alanbover/deathnode/monitor"
	"testing"
)

func TestRecommender(t *testing.T) {

	Convey("When creating a recommender", t, func() {

		monitor := prepareMonitors(&aws.ConnectionMock{
			Records: map[string]*[]string{
				"DescribeInstanceById": {"default", "default", "default"},
				"DescribeAGByName":     {"default"},
			},
		})
		Convey("it should raise an issue if the recommender doesn't exist", func() {
			_, err := newRecommender("noExistingRecommender")
			So(err, ShouldNotBeNil)
		})
		Convey("if it's of firstAvailableAgent type, if should return the first instance", func() {
			recommender, _ := newRecommender("firstAvailableAgent")
			instances := monitor.GetInstances()
			So(recommender.find(instances), ShouldEqual, instances[0])
		})
	})
}

func prepareMonitors(awsConn *aws.ConnectionMock) *monitor.AutoscalingGroupMonitor {

	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, []string{"some-Autoscaling-Group"}, "DEATH_NODE_MARK")
	autoscalingGroups.Refresh()

	return autoscalingGroups.GetAllMonitors()[0]
}
