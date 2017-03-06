package mesos

import (
	"encoding/json"
	"net/http"
	"fmt"
	"bytes"
	"io/ioutil"
	"text/template"
	"runtime"
	"path"
)

type MesosConnection struct {
	client		http.Client
	masterIP	string
}

type MesosConnectionInterface interface {
	GetMesosTasks() (*TasksResponse, error)
	GetMesosFrameworks() (*FrameworksResponse, error)
	GetMesosSlaves() (*SlavesResponse, error)
	SetHostInMaintenance(hostname, ip string) error
}

type SlavesResponse struct {
	Slaves	[]Slave 	`json:"slaves"`
}

type Slave struct {
	Id		string 	`json:"id"`
	Pid		string  `json:"pid"`
	Hostname	string  `json:"hostname"`
}

type FrameworksResponse struct {
	Frameworks	[]Framework 	`json:"frameworks"`
}

type Framework struct {
	Id		string 		`json:"id"`
	Name		string		`json:"name"`
	Active		bool		`json:"active"`
}

type TasksResponse struct {
	Tasks		[]Task `json:"tasks"`
}

type Task struct {
	Name		string 		`json:"name"`
	State 		string 		`json:"state"`
	Slave_id	string 		`json:"slave_id"`
	Framework_id	string 		`json:"framework_id"`
	Statuses	[]Status	`json:"statuses"`
}

type Status struct {
	State		string 		`json:"state"`
	Timestamp	float64  	`json:"timestamp"`
}

type hostInMaintenanceRequest struct {
	Hostname	string
	Ip		string
}

func (c* MesosConnection) GetMesosTasks() (*TasksResponse, error) {

	url := fmt.Sprintf(c.masterIP + "/master/tasks")

	var tasks TasksResponse
	mesos_get_api_call(url, &tasks)

	return &tasks, nil
}

func (c* MesosConnection) GetMesosFrameworks() (*FrameworksResponse, error) {

	url := fmt.Sprintf(c.masterIP + "/master/frameworks")

	var frameworks FrameworksResponse
	mesos_get_api_call(url, &frameworks)

	return &frameworks, nil
}

func (c* MesosConnection) GetMesosSlaves() (*SlavesResponse, error) {

	url := fmt.Sprintf(c.masterIP + "/master/slaves")

	var slaves SlavesResponse
	mesos_get_api_call(url, &slaves)

	return &slaves, nil
}

func (c* MesosConnection) SetHostInMaintenance(hostname, ip string) error {
	url := fmt.Sprintf(c.masterIP + "/maintenance/schedule")
	template_path := getCurrentPath() + "/templates/maintenance.tmpl"

	var payload bytes.Buffer
	request := &hostInMaintenanceRequest{
		Hostname: hostname,
		Ip: ip,
	}

	parse_template(template_path,  payload, request)

	mesos_post_api_call(url, payload.Bytes())
	return nil
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
		return nil
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return nil
	}

	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Print("Error decoding HTTP request: ", err)
	}

	return nil
}

func mesos_post_api_call(url string, payload []byte) error {
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Print("Error calling HTTP request: ", err)
		return nil
	}

	defer resp.Body.Close()
	return nil
}

func getCurrentPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}
