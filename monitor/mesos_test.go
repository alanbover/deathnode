package monitor

import (
	"encoding/json"
	"fmt"
	"github.com/alanbover/deathnode/mesos"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestGetMesosSlaveIdByIp(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("", "")
		monitor.Refresh()

		Convey("GetMesosSlaveByIp should return an slave it if exists", func() {
			mesosAgent, _ := monitor.GetMesosAgentByIP("10.0.0.2")
			So(mesosAgent.ID, ShouldEqual, "mesosslave1")
		})
		Convey("GetMesosSlaveByIp should return an error if it doesn't exists", func() {
			_, err := monitor.GetMesosAgentByIP("10.0.0.10")
			So(err, ShouldNotBeNil)
		})
	})
}

func TestGetMesosFrameworks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("frameworkName1", "")

		Convey("getProtectedFrameworks should return only the ones that match the protected frameworks", func() {
			frameworks := monitor.getProtectedFrameworks()
			So(len(frameworks), ShouldEqual, 1)
			So(frameworks, ShouldContainKey, "frameworkId1")
		})
	})
}

func TestHasProtectedFrameworksTasks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("frameworkName1", "")
		monitor.Refresh()

		Convey("HasProtectedFrameworksTasks returns", func() {
			Convey("true if a node have tasks running from protected frameworks", func() {
				value, _ := monitor.HasProtectedFrameworksTasks("10.0.0.2")
				So(value, ShouldBeTrue)
			})
			Convey("false if a node doesn't have tasks running from protected frameworks", func() {
				value, _ := monitor.HasProtectedFrameworksTasks("10.0.0.4")
				So(value, ShouldBeFalse)
			})
		})
	})
}

func TestHasProtectedLabelTasks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("", "DEATHNODE_PROTECTED")
		monitor.Refresh()

		Convey("HasProtectedLabelTasks returns", func() {
			Convey("true if a node have tasks running from protected labels", func() {
				value, _ := monitor.HasProtectedLabelTasks("10.0.0.2")
				So(value, ShouldBeTrue)
			})
			Convey("false if a node doesn't have tasks running from protected labels", func() {
				value, _ := monitor.HasProtectedLabelTasks("10.0.0.4")
				So(value, ShouldBeFalse)
			})
		})
	})
}

func TestSetMesosAgentsInMaintenance(t *testing.T) {
	Convey("When generating the payload for a maintenance call", t, func() {
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{},
		}
		templateJSON := mesos.MaintenanceRequest{}
		var testValues = []struct {
			hosts map[string]string
			num   int
		}{
			{map[string]string{}, 0},
			{map[string]string{"hostname1": "10.0.0.1"}, 1},
			{map[string]string{"hostname1": "10.0.0.1", "hostname2": "10.0.0.2"}, 2},
		}

		for _, testValue := range testValues {
			Convey(fmt.Sprintf("it should be possible to configure for %v agents", testValue.num), func() {
				template := mesosConn.GenMaintenanceCallPayload(testValue.hosts)
				json.Unmarshal(template, &templateJSON)
				So(len(templateJSON.Windows[0].MachinesIds), ShouldEqual, testValue.num)
			})
		}
	})
}

func createTestMesosMonitor(protectedFramework string, protectedTasksLabels string) *MesosMonitor {
	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks":              {"default"},
			"GetMesosSlaves":                  {"default"},
			"GetMesosTasks":                   {"default"},
			"GetMesosTasksWithLabelProtected": {"default"},
		},
	}
	return NewMesosMonitor(mesosConn, []string{protectedFramework}, []string{protectedTasksLabels})
}
