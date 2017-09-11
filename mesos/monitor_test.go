package mesos

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMesosSlaveIdByIp(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		mesosConn := &ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		mesosMonitor := NewMonitor(mesosConn, []string{""})
		mesosMonitor.Refresh()

		Convey("GetMesosSlaveByIp should return an slave it if exists", func() {
			mesosAgent, _ := mesosMonitor.GetMesosAgentByIP("10.0.0.2")
			So(mesosAgent.ID, ShouldEqual, "mesosslave1")
		})
		Convey("GetMesosSlaveByIp should return an error if it doesn't exists", func() {
			_, err := mesosMonitor.GetMesosAgentByIP("10.0.0.10")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetMesosFrameworks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		mesosConn := &ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		monitor := NewMonitor(mesosConn, []string{"frameworkName1"})

		Convey("getProtectedFrameworks should return only the ones that match the protected frameworks", func() {
			frameworks := monitor.getProtectedFrameworks()
			So(len(frameworks), ShouldEqual, 1)
			So(frameworks, ShouldContainKey, "frameworkId1")
		})
	})
}

func TestHasProtectedFrameworksTasks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		mesosConn := &ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		}
		monitor := NewMonitor(mesosConn, []string{"frameworkName1"})
		monitor.Refresh()

		Convey("HasProtectedFrameworksTasks returns", func() {
			Convey("true if a node have tasks running from protected frameworks", func() {
				So(monitor.HasProtectedFrameworksTasks("10.0.0.2"), ShouldBeTrue)
			})
			Convey("false if a node doesn't have tasks running from protected frameworks", func() {
				So(monitor.HasProtectedFrameworksTasks("10.0.0.4"), ShouldBeFalse)
			})
		})
	})
}