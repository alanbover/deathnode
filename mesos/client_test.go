package mesos

import (
	"encoding/json"
	"testing"
)

func TestGenerateTemplate(t *testing.T) {

	hosts := map[string]string{}
	hosts["hostname1"] = "10.0.0.1"
	hosts["hostname2"] = "10.0.0.2"
	template, _ := generateTemplate(hosts)
	templateJSON := MaintenanceRequest{}
	json.Unmarshal(template, &templateJSON)
	if len(templateJSON.Windows[0].MachinesIds) != 2 {
		t.Fatal("Template generated incorrectly")
	}
}
