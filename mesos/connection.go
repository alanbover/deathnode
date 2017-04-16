package mesos

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
)

type MesosConnectionInterface interface {
	getMesosTasks() (*TasksResponse, error)
	getMesosFrameworks() (*FrameworksResponse, error)
	getMesosSlaves() (*SlavesResponse, error)
	setHostInMaintenance(hostname, ip string) error
}

type MesosConnection struct {
	MasterUrl string
}

type SlavesResponse struct {
	Slaves []Slave `json:"slaves"`
}

type Slave struct {
	Id       string `json:"id"`
	Pid      string `json:"pid"`
	Hostname string `json:"hostname"`
}

type FrameworksResponse struct {
	Frameworks []Framework `json:"frameworks"`
}

type Framework struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type TasksResponse struct {
	Tasks []Task `json:"tasks"`
}

type Task struct {
	Name         string   `json:"name"`
	State        string   `json:"state"`
	Slave_id     string   `json:"slave_id"`
	Framework_id string   `json:"framework_id"`
	Statuses     []Status `json:"statuses"`
}

type Status struct {
	State     string  `json:"state"`
	Timestamp float64 `json:"timestamp"`
}

type hostInMaintenanceRequest struct {
	Hostname string
	Ip       string
}

func (c *MesosConnection) setHostInMaintenance(hostname, ip string) error {

	url := fmt.Sprintf(c.MasterUrl + "/maintenance/schedule")
	template_path := getCurrentPath() + "/templates/maintenance.tmpl"

	var payload bytes.Buffer
	request := &hostInMaintenanceRequest{
		Hostname: hostname,
		Ip:       ip,
	}

	parse_template(template_path, payload, request)

	mesos_post_api_call(url, payload.Bytes())
	return nil
}

func (c *MesosConnection) getMesosTasks() (*TasksResponse, error) {

	url := fmt.Sprintf(c.MasterUrl + "/master/tasks")

	var tasks TasksResponse
	mesos_get_api_call(url, &tasks)

	return &tasks, nil
}

func (c *MesosConnection) getMesosFrameworks() (*FrameworksResponse, error) {

	url := fmt.Sprintf(c.MasterUrl + "/master/frameworks")

	var frameworks FrameworksResponse
	mesos_get_api_call(url, &frameworks)

	return &frameworks, nil
}

func (c *MesosConnection) getMesosSlaves() (*SlavesResponse, error) {

	url := fmt.Sprintf(c.MasterUrl + "/master/slaves")

	var slaves SlavesResponse
	mesos_get_api_call(url, &slaves)

	return &slaves, nil
}

func parse_template(template_path string, doc bytes.Buffer, values interface{}) error {

	maintenance_template, _ := ioutil.ReadFile(template_path)
	tmpl, _ := template.New("template").Parse(string(maintenance_template))
	err := tmpl.Execute(&doc, values)
	return err
}

func mesos_get_api_call(url string, response interface{}) error {

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

func mesos_post_api_call(url string, payload []byte) error {

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
