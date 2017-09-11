package mesos

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"fmt"
	"strings"
)

// Monitor monitors the mesos cluster, creating a cache to reduce the number of calls against it
type Monitor struct {
	mesosConn           ClientInterface
	mesosCache          *mesosCache
	protectedFrameworks []string
}

// MesosCache stores the objects of the mesosApi in a way that is directly accesible
// tasks: map[slaveId][]Task
// frameworks: map[frameworkID]Framework
// slaves: map[privateIPAddress]Slave
type mesosCache struct {
	tasks      map[string][]Task
	frameworks map[string]Framework
	slaves     map[string]Slave
}

// NewMonitor returns a new mesos.monitor object
func NewMonitor(mesosConn ClientInterface, protectedFrameworks []string) *Monitor {

	return &Monitor{
		mesosConn: mesosConn,
		mesosCache: &mesosCache{
			tasks:      map[string][]Task{},
			frameworks: map[string]Framework{},
			slaves:     map[string]Slave{},
		},
		protectedFrameworks: protectedFrameworks,
	}
}

// Refresh updates the mesos cache
func (m *Monitor) Refresh() {

	m.mesosCache.tasks = m.getTasks()
	m.mesosCache.frameworks = m.getProtectedFrameworks()
	m.mesosCache.slaves = m.getSlaves()
}

func (m *Monitor) getProtectedFrameworks() map[string]Framework {

	frameworksMap := map[string]Framework{}
	frameworksResponse, _ := m.mesosConn.getMesosFrameworks()
	for _, framework := range frameworksResponse.Frameworks {
		for _, protectedFramework := range m.protectedFrameworks {
			if protectedFramework == framework.Name {
				frameworksMap[framework.ID] = framework
			}
		}
	}
	return frameworksMap
}

func (m *Monitor) getSlaves() map[string]Slave {

	slavesMap := map[string]Slave{}
	slavesResponse, _ := m.mesosConn.getMesosSlaves()
	for _, slave := range slavesResponse.Slaves {
		ipAddress := m.getAgentIPAddressFromPID(slave.Pid)
		slavesMap[ipAddress] = slave
	}
	return slavesMap
}

func (m *Monitor) getAgentIPAddressFromPID(pid string) string {

	tmp := strings.Split(pid, "@")[1]
	return strings.Split(tmp, ":")[0]
}

func (m *Monitor) getTasks() map[string][]Task {

	tasksMap := map[string][]Task{}
	tasksResponse, _ := m.mesosConn.getMesosTasks()
	for _, task := range tasksResponse.Tasks {
		if task.State == "TASK_RUNNING" {
			tasksMap[task.SlaveID] = append(tasksMap[task.SlaveID], task)
		}
	}
	return tasksMap
}

// GetMesosSlaveByIP returns the Mesos slave that matches a certain IP
func (m *Monitor) GetMesosAgentByIP(ipAddress string) (Slave, error) {

	slave, ok := m.mesosCache.slaves[ipAddress]
	if ok {
		return slave, nil
	}

	return Slave{}, fmt.Errorf("Instance with ip %v not found in Mesos", ipAddress)
}

// SetMesosAgentsInMaintenance sets a list of mesos agents in Maintenance mode
func (m *Monitor) SetMesosAgentsInMaintenance(hosts map[string]string) error {
	return m.mesosConn.setHostsInMaintenance(hosts)
}

// HasProtectedFrameworksTasks returns true if the mesos agent has any tasks running from any of the
// protected frameworks.
func (m *Monitor) HasProtectedFrameworksTasks(ipAddress string) bool {

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
