package mesos

import (
	"fmt"
	"strings"
)

type MesosMonitor struct {
	mesosConn  MesosConnectionInterface
	mesosCache *mesosCache
}

type mesosCache struct {
	tasks      []Task
	frameworks []Framework
	slaves     []Slave
}

func NewMesosMonitor(mesosConn MesosConnectionInterface) *MesosMonitor {

	return &MesosMonitor{
		mesosConn: mesosConn,
		mesosCache: &mesosCache{
			tasks:      []Task{},
			frameworks: []Framework{},
			slaves:     []Slave{},
		},
	}
}

func (m *MesosMonitor) Refresh() {

	frameworksResponse, _ := m.mesosConn.getMesosFrameworks()
	tasksResponse, _ := m.mesosConn.getMesosTasks()
	slavesResponse, _ := m.mesosConn.getMesosSlaves()

	m.mesosCache.frameworks = frameworksResponse.Frameworks
	m.mesosCache.tasks = tasksResponse.Tasks
	m.mesosCache.slaves = slavesResponse.Slaves
}

func (m *MesosMonitor) GetMesosSlaveByIp(ipAddress string) (Slave, error) {

	for _, slave := range m.mesosCache.slaves {
		if strings.Contains(slave.Pid, ipAddress) {
			return slave, nil
		}
	}

	return Slave{}, fmt.Errorf("Instance with ip %v not found in Mesos", ipAddress)
}

func (m *MesosMonitor) SetMesosSlaveInMaintenance(hostname, ip string) {

	m.mesosConn.setHostInMaintenance(hostname, ip)
}

func (m *MesosMonitor) DoesSlaveHasFrameworks(ipAddress string, frameworks []string) (bool, error) {

	slave, err := m.GetMesosSlaveByIp(ipAddress)
	if err != nil {
		return false, err
	}
	slaveTasks := []Task{}
	for _, task := range m.mesosCache.tasks {
		if task.Slave_id == slave.Id {
			slaveTasks = append(slaveTasks, task)
		}
	}

	frameworkIds := []string{}
	for _, framework := range frameworks {
		for _, frameworkCache := range m.mesosCache.frameworks {
			if frameworkCache.Name == framework {
				frameworkIds = append(frameworkIds, frameworkCache.Name)
			}
		}
	}

	for _, task := range slaveTasks {
		for _, frameworkId := range frameworkIds {
			if task.Framework_id == frameworkId {
				return true, nil
			}
		}
	}

	return false, nil
}
