package mesos

import (
	"testing"
)

func TestGetMesosSlaveIdByIp(t *testing.T) {

	mesosConn := &ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default"},
			"GetMesosSlaves":     {"default"},
			"GetMesosTasks":      {"default"},
		},
	}

	protectedFrameworks := []string{"framework1"}
	mesosMonitor := NewMonitor(mesosConn, protectedFrameworks)
	mesosMonitor.Refresh()
	mesosAgentID, _ := mesosMonitor.GetMesosSlaveByIP("10.0.0.2")

	if mesosAgentID.ID != "mesosslave1" {
		t.Fatal("Incorrect Mesos slave Id")
	}
}
