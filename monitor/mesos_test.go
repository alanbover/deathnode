package monitor

import (
	"encoding/json"
	"fmt"
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/mesos"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

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

func TestIsProtected(t *testing.T) {

	Convey("when calling IsProtected", t, func() {
		Convey("when checking protected labels", func() {
			monitor := createTestMesosMonitor("", "DEATHNODE_PROTECTED")
			monitor.Refresh()
			Convey("true if a node have tasks running from protected labels", func() {
				So(monitor.IsProtected("10.0.0.2"), ShouldBeTrue)
			})
			Convey("false if a node doesn't have tasks running from protected labels", func() {
				So(monitor.IsProtected("10.0.0.4"), ShouldBeFalse)
			})
		})
		Convey("when checking protected frameworks", func() {
			monitor := createTestMesosMonitor("frameworkName1", "")
			monitor.Refresh()
			Convey("true if a node have tasks running from protected frameworks", func() {
				So(monitor.IsProtected("10.0.0.2"), ShouldBeTrue)
			})
			Convey("false if a node doesn't have tasks running from protected frameworks", func() {
				So(monitor.IsProtected("10.0.0.4"), ShouldBeFalse)
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

	ctx := &context.ApplicationContext{
		MesosConn: &mesos.ClientMock{
			Records: map[string]*[]string{
				"GetMesosFrameworks": {"default"},
				"GetMesosSlaves":     {"default"},
				"GetMesosTasks":      {"default"},
			},
		},
		Conf: context.ApplicationConf{
			ProtectedFrameworks:  []string{protectedFramework},
			ProtectedTasksLabels: []string{protectedTasksLabels},
		},
	}

	return NewMesosMonitor(ctx)
}
