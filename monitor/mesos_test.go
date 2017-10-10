package monitor

import (
	"testing"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/alanbover/deathnode/mesos"
	"encoding/json"
	"fmt"
)

func TestGetMesosFrameworks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("frameworkName1")

		Convey("getProtectedFrameworks should return only the ones that match the protected frameworks", func() {
			frameworks := monitor.getProtectedFrameworks()
			So(len(frameworks), ShouldEqual, 1)
			So(frameworks, ShouldContainKey, "frameworkId1")
		})
	})
}

func TestHasProtectedFrameworksTasks(t *testing.T) {

	Convey("When creating a new mesos monitor", t, func() {
		monitor := createTestMesosMonitor("frameworkName1")
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

func TestSetMesosAgentsInMaintenance(t *testing.T) {
	Convey("When generating the payload for a maintenance call", t, func() {
		mesosConn := &mesos.ClientMock{
			Records: map[string]*[]string{},
		}
		templateJSON := mesos.MaintenanceRequest{}
		var testValues = []struct {
			hosts  map[string]string
			num int
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

func createTestMesosMonitor(protectedFramework string) *MesosMonitor {
	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default"},
			"GetMesosSlaves":     {"default"},
			"GetMesosTasks":      {"default"},
		},
	}
	return NewMesosMonitor(mesosConn, []string{protectedFramework})
}