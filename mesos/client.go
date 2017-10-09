package mesos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

// ClientInterface is an interface for mesos api clients
type ClientInterface interface {
	GetMesosTasks() (*TasksResponse, error)
	GetMesosFrameworks() (*FrameworksResponse, error)
	GetMesosAgents() (*SlavesResponse, error)
	SetHostsInMaintenance(map[string]string) error
}

// Client implements a client for mesos api
type Client struct {
	MasterURL string
}

// SlavesResponse is part of the mesos slaves response API endpoint
type SlavesResponse struct {
	Slaves []Slave `json:"slaves"`
}

// Slave is part of the mesos slaves response API endpoint
type Slave struct {
	ID       string `json:"id"`
	Pid      string `json:"pid"`
	Hostname string `json:"hostname"`
}

// FrameworksResponse is part of the mesos frameworks response API endpoint
type FrameworksResponse struct {
	Frameworks []Framework `json:"frameworks"`
}

// Framework is part of the mesos frameworks response API endpoint
type Framework struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// TasksResponse is part of the mesos tasks response API endpoint
type TasksResponse struct {
	Tasks []Task `json:"tasks"`
}

// Task is part of the mesos tasks response API endpoint
type Task struct {
	Name        string   `json:"name"`
	State       string   `json:"state"`
	SlaveID     string   `json:"slave_id"`
	FrameworkID string   `json:"framework_id"`
	Statuses    []Status `json:"statuses"`
	Labels      []Labels `json:"labels"`
}

type Labels struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Status is part of the mesos tasks response API endpoint
type Status struct {
	State     string  `json:"state"`
	Timestamp float64 `json:"timestamp"`
}

// MaintenanceRequest implements the payload for set mesos instances in maintenance API call
type MaintenanceRequest struct {
	Windows []MaintenanceWindow `json:"windows"`
}

// MaintenanceWindow implements the payload for set mesos instances in maintenance API call
type MaintenanceWindow struct {
	MachinesIds    []MaintenanceMachinesID   `json:"machine_ids"`
	Unavailability MaintenanceUnavailability `json:"unavailability"`
}

// MaintenanceMachinesID implements the payload for set mesos instances in maintenance API call
type MaintenanceMachinesID struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

// MaintenanceUnavailability implements the payload for set mesos instances in maintenance API call
type MaintenanceUnavailability struct {
	Start MaintenanceStart `json:"start"`
}

// MaintenanceStart implements the payload for set mesos instances in maintenance API call
type MaintenanceStart struct {
	Nanoseconds int32 `json:"nanoseconds"`
}

// SetHostsInMaintenance configures nodes in maintenance for Mesos cluster
func (c *Client) SetHostsInMaintenance(hosts map[string]string) error {

	url := fmt.Sprintf(c.MasterURL + "/maintenance/schedule")

	payload := genMaintenanceCallPayload(hosts)

	err := mesosPostAPICall(url, payload)
	return err
}

// GetMesosTasks return the running tasks on the Mesos cluster
func (c *Client) GetMesosTasks() (*TasksResponse, error) {

	var tasks TasksResponse
	c.getMesosTasksRecursive(&tasks, 0)

	return &tasks, nil
}

func (c *Client) getMesosTasksRecursive(tasksResponse *TasksResponse, offset int) error {

	var tasks TasksResponse
	url := fmt.Sprintf("%s/master/tasks?limit=100&offset=%d", c.MasterURL, offset)
	err := mesosGetAPICall(url, &tasks)
	if err != nil {
		return err
	}

	tasksResponse.Tasks = append(tasksResponse.Tasks, tasks.Tasks...)

	if len(tasks.Tasks) == 100 {
		c.getMesosTasksRecursive(tasksResponse, offset+100)
	}

	return nil
}

// GetMesosFrameworks returns the registered frameworks in Mesos
func (c *Client) GetMesosFrameworks() (*FrameworksResponse, error) {

	url := fmt.Sprintf(c.MasterURL + "/master/frameworks")

	var frameworks FrameworksResponse
	mesosGetAPICall(url, &frameworks)

	return &frameworks, nil
}

// GetMesosAgents returns the Mesos Agents registered in the Mesos cluster
func (c *Client) GetMesosAgents() (*SlavesResponse, error) {

	url := fmt.Sprintf(c.MasterURL + "/master/slaves")

	var slaves SlavesResponse
	mesosGetAPICall(url, &slaves)

	return &slaves, nil
}

func genMaintenanceCallPayload(hosts map[string]string) []byte {

	maintenanceMachinesIDs := []MaintenanceMachinesID{}
	for host := range hosts {
		maintenanceMachinesID := MaintenanceMachinesID{
			Hostname: host,
			IP:       hosts[host],
		}
		maintenanceMachinesIDs = append(maintenanceMachinesIDs, maintenanceMachinesID)
	}

	maintenanceWindow := MaintenanceWindow{
		MachinesIds: maintenanceMachinesIDs,
		Unavailability: MaintenanceUnavailability{
			MaintenanceStart{
				Nanoseconds: 1,
			},
		},
	}

	maintenanceRequest := MaintenanceRequest{
		Windows: []MaintenanceWindow{maintenanceWindow},
	}

	template, _ := json.Marshal(maintenanceRequest)
	return template
}

func mesosGetAPICall(url string, response interface{}) error {

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Print("Error preparing HTTP request: ", err)
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return err
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Print("Error decoding HTTP request: ", err)
		return err
	}

	return nil
}

func mesosPostAPICall(url string, payload []byte) error {

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Print("Error preparing HTTP request: ", err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return err
	}

	defer resp.Body.Close()
	return nil
}

func getCurrentPath() string {

	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src/github.com/alanbover/deathnode/mesos")
}
