package monitor

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
	"regexp"
	"strings"
)

// MesosMonitor monitors the mesos cluster, creating a cache to reduce the number of calls against it
type MesosMonitor struct {
	mesosCache *mesosCache
	ctx        *context.ApplicationContext
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

// NewMesosMonitor returns a new mesos.monitor object
func NewMesosMonitor(ctx *context.ApplicationContext) *MesosMonitor {

	return &MesosMonitor{
		mesosCache: &mesosCache{
			tasks:      map[string][]mesos.Task{},
			frameworks: map[string]mesos.Framework{},
			slaves:     map[string]mesos.Slave{},
		},
		ctx: ctx,
	}
}

// Refresh updates the mesos cache
func (m *MesosMonitor) Refresh() {

	m.mesosCache.tasks = m.getTasks()
	m.mesosCache.frameworks = m.getProtectedFrameworks()
	m.mesosCache.slaves = m.getSlaves()
}

func (m *MesosMonitor) getProtectedFrameworks() map[string]mesos.Framework {

	protectedFrameworksMap := map[string]mesos.Framework{}
	response, err := m.ctx.MesosConn.GetMesosFrameworks()
	if err != nil {
		log.Warning(err)
		return protectedFrameworksMap
	}

	for _, framework := range response.Frameworks {
		for _, protectedFramework := range m.ctx.Conf.ProtectedFrameworks {
			if protectedFramework == framework.Name {
				protectedFrameworksMap[framework.ID] = framework
			}
		}
	}
	return protectedFrameworksMap
}

func (m *MesosMonitor) getSlaves() map[string]mesos.Slave {

	slavesMap := map[string]mesos.Slave{}
	response, err := m.ctx.MesosConn.GetMesosAgents()
	if err != nil {
		log.Warning(err)
		return slavesMap
	}

	for _, slave := range response.Slaves {
		ipAddress := m.getAgentIPAddressFromPID(slave.Pid)
		slavesMap[ipAddress] = slave
	}
	return slavesMap
}

func (m *MesosMonitor) getAgentIPAddressFromPID(pid string) string {

	tmp := strings.Split(pid, "@")[1]
	return strings.Split(tmp, ":")[0]
}

func (m *MesosMonitor) isTaskProtected(task mesos.Task) bool {

	for _, label := range task.Labels {
		for _, protectedTasksLabel := range m.ctx.Conf.ProtectedTasksLabels {
			if label.Key == protectedTasksLabel && strings.ToUpper(label.Value) == "TRUE" {
				return true
			}
		}
	}
	return false
}

func (m *MesosMonitor) getTasks() map[string][]mesos.Task {

	tasksMap := map[string][]mesos.Task{}
	response, err := m.ctx.MesosConn.GetMesosTasks()
	if err != nil {
		log.Warning(err)
		return tasksMap
	}

	for _, task := range response.Tasks {
		if task.State == "TASK_RUNNING" {
			task.IsProtected = m.isTaskProtected(task)
			tasksMap[task.SlaveID] = append(tasksMap[task.SlaveID], task)
		}
	}
	return tasksMap
}

// SetMesosAgentsInMaintenance sets a list of mesos agents in Maintenance mode
func (m *MesosMonitor) SetMesosAgentsInMaintenance(hosts map[string]string) error {
	return m.ctx.MesosConn.SetHostsInMaintenance(hosts)
}

func (m *MesosMonitor) isFromProtectedFramework(task mesos.Task) bool {

	framework, ok := m.mesosCache.frameworks[task.FrameworkID]
	if ok {
		log.Debugf("Framework %s is running on node %s, preventing Deathnode for killing it",
			framework.Name, task.SlaveID)
		return true
	}

	return false
}

func (m *MesosMonitor) hasProtectedLabel(task mesos.Task) bool {

	if task.IsProtected {
		log.Debugf("Protected task %s is running on node %s, preventing Deathnode for killing it",
			task.Name, task.SlaveID)
		return true
	}
	return false
}

type taskEvaluate func(*MesosMonitor, mesos.Task) bool

func (m *MesosMonitor) agentTaskEvaluation(ipAddress string, fn taskEvaluate) bool {

	slaveID := m.mesosCache.slaves[ipAddress].ID
	slaveTasks := m.mesosCache.tasks[slaveID]
	for _, task := range slaveTasks {
		if fn(m, task) {
			return true
		}
	}
	return false
}

// HasTaskNameMatchRegexp returns true if the host has any taskName that match a certain regexp
func (m *MesosMonitor) HasTaskNameMatchRegexp(ipAddress, taskRegexp string) bool {

	return m.agentTaskEvaluation(ipAddress, func(m *MesosMonitor, task mesos.Task) bool {
		matched, _ := regexp.MatchString(taskRegexp, task.Name)
		return matched
	})
}

// HasFrameworks returns true if the host has any task from any of the frameworks
func (m *MesosMonitor) HasFrameworks(ipAddress, framework string) bool {

	return m.agentTaskEvaluation(ipAddress, func(m *MesosMonitor, task mesos.Task) bool {
		return framework == m.mesosCache.frameworks[task.FrameworkID].Name
	})
}

// IsProtected returns true if the mesos agent has any protected condition.
func (m *MesosMonitor) IsProtected(ipAddress string) bool {

	return m.agentTaskEvaluation(ipAddress, func(m *MesosMonitor, task mesos.Task) bool {
		return m.hasProtectedLabel(task) || m.isFromProtectedFramework(task)
	})
}
