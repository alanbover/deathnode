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

	protectedFrameworks := []string{"framework1"}
	mesosMonitor := NewMesosMonitor(mesosConn, protectedFrameworks)
	mesosMonitor.Refresh()
	mesosAgentId, _ := mesosMonitor.GetMesosSlaveByIp("10.0.0.2")

	if mesosAgentId.Id != "mesosslave1" {
		t.Fatal("Incorrect Mesos slave Id")
	}
}
