package monitor

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"fmt"
	"strings"
	"github.com/alanbover/deathnode/mesos"
)

// Monitor monitors the mesos cluster, creating a cache to reduce the number of calls against it
type MesosMonitor struct {
	mesosConn           mesos.ClientInterface
	mesosCache          *mesosCache
	protectedFrameworks []string
}

// MesosCache stores the objects of the mesosApi in a way that is directly accesible
// tasks: map[slaveId][]Task
// frameworks: map[frameworkID]Framework
// slaves: map[privateIPAddress]Slave
type mesosCache struct {
	tasks      map[string][]mesos.Task
	frameworks map[string]mesos.Framework
	slaves     map[string]mesos.Slave
}

// NewMonitor returns a new mesos.monitor object
func NewMesosMonitor(mesosConn mesos.ClientInterface, protectedFrameworks []string) *MesosMonitor {

	return &MesosMonitor{
		mesosConn: mesosConn,
		mesosCache: &mesosCache{
			tasks:      map[string][]mesos.Task{},
			frameworks: map[string]mesos.Framework{},
			slaves:     map[string]mesos.Slave{},
		},
		protectedFrameworks: protectedFrameworks,
	}
}

// Refresh updates the mesos cache
func (m *MesosMonitor) Refresh() {

	m.mesosCache.tasks = m.getTasks()
	m.mesosCache.frameworks = m.getProtectedFrameworks()
	m.mesosCache.slaves = m.getSlaves()
}

func (m *MesosMonitor) getProtectedFrameworks() map[string]mesos.Framework {

	frameworksMap := map[string]mesos.Framework{}
	frameworksResponse, _ := m.mesosConn.GetMesosFrameworks()
	for _, framework := range frameworksResponse.Frameworks {
		for _, protectedFramework := range m.protectedFrameworks {
			if protectedFramework == framework.Name {
				frameworksMap[framework.ID] = framework
			}
		}
	}
	return frameworksMap
}

func (m *MesosMonitor) getSlaves() map[string]mesos.Slave {

	slavesMap := map[string]mesos.Slave{}
	slavesResponse, _ := m.mesosConn.GetMesosSlaves()
	for _, slave := range slavesResponse.Slaves {
		ipAddress := m.getAgentIPAddressFromPID(slave.Pid)
		slavesMap[ipAddress] = slave
	}
	return slavesMap
}

func (m *MesosMonitor) getAgentIPAddressFromPID(pid string) string {

	tmp := strings.Split(pid, "@")[1]
	return strings.Split(tmp, ":")[0]
}

func (m *MesosMonitor) getTasks() map[string][]mesos.Task {

	tasksMap := map[string][]mesos.Task{}
	tasksResponse, _ := m.mesosConn.GetMesosTasks()
	for _, task := range tasksResponse.Tasks {
		if task.State == "TASK_RUNNING" {
			tasksMap[task.SlaveID] = append(tasksMap[task.SlaveID], task)
		}
	}
	return tasksMap
}

// GetMesosSlaveByIP returns the Mesos slave that matches a certain IP
func (m *MesosMonitor) GetMesosAgentByIP(ipAddress string) (mesos.Slave, error) {

	slave, ok := m.mesosCache.slaves[ipAddress]
	if ok {
		return slave, nil
	}

	return mesos.Slave{}, fmt.Errorf("Instance with ip %v not found in Mesos", ipAddress)
}

// SetMesosAgentsInMaintenance sets a list of mesos agents in Maintenance mode
func (m *MesosMonitor) SetMesosAgentsInMaintenance(hosts map[string]string) error {
	return m.mesosConn.SetHostsInMaintenance(hosts)
}

// HasProtectedFrameworksTasks returns true if the mesos agent has any tasks running from any of the
// protected frameworks.
func (m *MesosMonitor) HasProtectedFrameworksTasks(ipAddress string) bool {

	slaveID := m.mesosCache.slaves[ipAddress].ID
	fmt.Println(m.mesosCache.slaves)
	slaveTasks := m.mesosCache.tasks[slaveID]
	for _, task := range slaveTasks {
		_, ok := m.mesosCache.frameworks[task.FrameworkID]
		if ok {
			return true
		}
	}

	return false
}
