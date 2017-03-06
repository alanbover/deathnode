package mesos

import (
	"encoding/json"
	"net/http"
	"fmt"
	"bytes"
	//"io/ioutil"
	//"text/template"
	"runtime"
	"path"
)

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

type MesosConnection struct {
	client		http.Client
	masterIP	string
}

func (c* MesosConnection) GetMesosTasks() error {

	url := fmt.Sprintf(c.masterIP + "/master/tasks")

	var tasks TasksResponse
	mesos_get_api_call(url, &tasks)

	fmt.Println(tasks)

	return nil
}

func (c* MesosConnection) GetMesosFrameworks() error {

	url := fmt.Sprintf(c.masterIP + "/master/frameworks")

	var frameworks FrameworksResponse
	mesos_get_api_call(url, &frameworks)

	fmt.Println(frameworks)

	return nil
}

func (c* MesosConnection) GetMesosSlaves() error {

	url := fmt.Sprintf(c.masterIP + "/master/slaves")

	var slaves SlavesResponse
	mesos_get_api_call(url, &slaves)

	fmt.Println(slaves)

	return nil
}

func (c* MesosConnection) SetHostInMaintenance() error {
	//url := fmt.Sprintf(c.masterIP + "/maintenance/schedule")

	//maintenance_template, err := ioutil.ReadFile(getCurrentPath() + "/templates/maintenance.tmpl")

	//tmpl, err := template.New("test").Parse(maintenance_template)

	//var payload string

	//mesos_post_api_call(url, payload)

	return nil
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
