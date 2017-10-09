package monitor

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"fmt"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
	"strings"
)

// MesosMonitor monitors the mesos cluster, creating a cache to reduce the number of calls against it
type MesosMonitor struct {
	mesosConn            mesos.ClientInterface
	mesosCache           *mesosCache
	protectedFrameworks  []string
	protectedTasksLabels []string
}

// MesosCache stores the objects of the mesosApi in a way that is directly accesible
// tasks: map[slaveId][]Task
// frameworks: map[frameworkID]Framework
// slaves: map[privateIPAddress]Slave
type mesosCache struct {
	tasks                   map[string][]mesos.Task
	frameworks              map[string]mesos.Framework
	slaves                  map[string]mesos.Slave
	tasksWithLabelProtected map[string][]mesos.Task
}

// NewMesosMonitor returns a new mesos.monitor object
func NewMesosMonitor(mesosConn mesos.ClientInterface, protectedFrameworks []string, protectedTasksLabels []string) *MesosMonitor {

	return &MesosMonitor{
		mesosConn: mesosConn,
		mesosCache: &mesosCache{
			tasks:                   map[string][]mesos.Task{},
			frameworks:              map[string]mesos.Framework{},
			slaves:                  map[string]mesos.Slave{},
			tasksWithLabelProtected: map[string][]mesos.Task{},
		},
		protectedFrameworks:  protectedFrameworks,
		protectedTasksLabels: protectedTasksLabels,
	}
}

// Refresh updates the mesos cache
func (m *MesosMonitor) Refresh() {

	m.mesosCache.tasks, m.mesosCache.tasksWithLabelProtected = m.getTasks()

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
	slavesResponse, _ := m.mesosConn.GetMesosAgents()
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

func (m *MesosMonitor) getTasks() (map[string][]mesos.Task, map[string][]mesos.Task) {

	tasksMap := map[string][]mesos.Task{}
	tasksWithLabelProtectedMap := map[string][]mesos.Task{}
	tasksResponse, _ := m.mesosConn.GetMesosTasks()
	for _, task := range tasksResponse.Tasks {
		if task.State == "TASK_RUNNING" {
			tasksMap[task.SlaveID] = append(tasksMap[task.SlaveID], task)
			for _, label := range task.Labels {
				for _, protectedTasksLabel := range m.protectedTasksLabels {
					if label.Key == protectedTasksLabel && strings.ToUpper(label.Value) == "TRUE" {
						tasksWithLabelProtectedMap[task.Name] = append(tasksWithLabelProtectedMap[task.Name], task)
					}
				}
			}
		}
	}
	return tasksMap, tasksWithLabelProtectedMap
}

// GetMesosAgentByIP returns the Mesos slave that matches a certain IP
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
func (m *MesosMonitor) HasProtectedFrameworksTasks(ipAddress string) (bool, *mesos.Framework) {

	slaveID := m.mesosCache.slaves[ipAddress].ID
	fmt.Println(m.mesosCache.slaves)
	slaveTasks := m.mesosCache.tasks[slaveID]
	for _, task := range slaveTasks {
		framework, ok := m.mesosCache.frameworks[task.FrameworkID]
		if ok {
			return true, &framework
		}
	}

	return false, nil
}

// HasProtectedLabelsTasks returns true if the mesos agent has any label tasks running from any of the
// protected label.
func (m *MesosMonitor) HasProtectedLabelTasks(ipAddress string) (bool, *mesos.Task) {

	slaveID := m.mesosCache.slaves[ipAddress].ID
	fmt.Println(m.mesosCache.slaves)
	slaveTasks := m.mesosCache.tasks[slaveID]
	for _, task := range slaveTasks {
		log.Infof("Searching Task %v removed. Deleting it", task)
		_, ok := m.mesosCache.tasksWithLabelProtected[task.Name]
		if ok {
			return true, &task
		}
	}

	return false, nil
}

func (m *MesosMonitor) IsProtected(ipAddress string) (bool, string) {
	if hasProtectedTasks, task := m.HasProtectedLabelTasks(ipAddress); hasProtectedTasks {
		return hasProtectedTasks, fmt.Sprintf("Task %s is protected, preventing Deathnode for killing it", task.Name)
	}
	if hasProtectedFrameworks, framework := m.HasProtectedFrameworksTasks(ipAddress); hasProtectedFrameworks {
		return hasProtectedFrameworks, fmt.Sprintf("Framework %s is running on node, preventing Deathnode for killing it", framework.Name)
	}
	return false, ""
}
