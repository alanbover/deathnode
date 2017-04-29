package mesos

// Monitor holds a connection to mesos, and a cache for every iteration
// With MesosCache we reduce the number of calls to mesos, also we map it for quicker access

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"strings"
)

type MesosMonitor struct {
	mesosConn           MesosConnectionInterface
	mesosCache          *mesosCache
	protectedFrameworks []string
}

type mesosCache struct {
	tasks      map[string][]Task
	frameworks map[string]Framework
	slaves     map[string]Slave
}

func NewMesosMonitor(mesosConn MesosConnectionInterface, protectedFrameworks []string) *MesosMonitor {

	return &MesosMonitor{
		mesosConn: mesosConn,
		mesosCache: &mesosCache{
			tasks:      map[string][]Task{},
			frameworks: map[string]Framework{},
			slaves:     map[string]Slave{},
		},
		protectedFrameworks: protectedFrameworks,
	}
}

func (m *MesosMonitor) Refresh() {

	m.mesosCache.tasks = m.getMesosTasks()
	m.mesosCache.frameworks = m.getMesosFrameworks()
	m.mesosCache.slaves = m.getMesosSlaves()
}

func (m *MesosMonitor) getMesosFrameworks() map[string]Framework {

	frameworksMap := map[string]Framework{}
	frameworksResponse, _ := m.mesosConn.getMesosFrameworks()
	for _, framework := range frameworksResponse.Frameworks {
		for _, protectedFramework := range m.protectedFrameworks {
			if protectedFramework == framework.Name {
				log.Debugf("Add framework id %s to mesos cache", framework.Id)
				frameworksMap[framework.Id] = framework
			}
		}
	}
	return frameworksMap
}

func (m *MesosMonitor) getMesosSlaves() map[string]Slave {

	slavesMap := map[string]Slave{}
	slavesResponse, _ := m.mesosConn.getMesosSlaves()
	for _, slave := range slavesResponse.Slaves {
		ipAddress := m.getIpAddressFromSlavePID(slave.Pid)
		log.Debugf("Add slave %s to mesos cache", ipAddress)
		slavesMap[ipAddress] = slave
	}
	return slavesMap
}

func (m *MesosMonitor) getIpAddressFromSlavePID(pid string) string {

	tmp := strings.Split(pid, "@")[1]
	ipAddress := strings.Split(tmp, ":")[0]
	return ipAddress

}

func (m *MesosMonitor) getMesosTasks() map[string][]Task {

	tasksMap := map[string][]Task{}
	tasksResponse, _ := m.mesosConn.getMesosTasks()
	for _, task := range tasksResponse.Tasks {
		if task.State == "TASK_RUNNING" {
			log.Debugf("Add task %s to to host %s", task.Name, task.Slave_id)
			tasksMap[task.Slave_id] = append(tasksMap[task.Slave_id], task)
		}
	}
	return tasksMap
}

func (m *MesosMonitor) GetMesosSlaveByIp(ipAddress string) (Slave, error) {

	slave, ok := m.mesosCache.slaves[ipAddress]
	if ok {
		return slave, nil
	}

	return Slave{}, fmt.Errorf("Instance with ip %v not found in Mesos", ipAddress)
}

func (m *MesosMonitor) SetMesosSlavesInMaintenance(hosts map[string]string) error {
	return m.mesosConn.setHostsInMaintenance(hosts)
}

func (m *MesosMonitor) DoesSlaveHasFrameworks(ipAddress string) (bool, error) {

	slaveId := m.mesosCache.slaves[ipAddress].Id
	slaveTasks := m.mesosCache.tasks[slaveId]
	for _, task := range slaveTasks {
		log.Debugf("Check if framework id %s is protected", task.Framework_id)
		_, ok := m.mesosCache.frameworks[task.Framework_id]
		if ok {
			return true, nil
		}
	}

	return false, fmt.Errorf("Instance with ip %v not found in Mesos", ipAddress)
}
