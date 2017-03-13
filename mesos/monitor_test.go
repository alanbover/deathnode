package mesos

import (
	"testing"
)

func TestGetMesosSlaveIdByIp(t *testing.T) {

	mesosConn := &MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default"},
			"GetMesosSlaves":     &[]string{"default"},
			"GetMesosTasks":      &[]string{"default"},
		},
	}

	mesosMonitor := NewMesosMonitor(mesosConn)
	mesosMonitor.Refresh()
	mesosAgentId, _ := mesosMonitor.GetMesosSlaveByIp("10.0.0.2")

	if mesosAgentId.Id != "mesosslave2" {
		t.Fatal("Incorrect Mesos slave Id")
	}
}
