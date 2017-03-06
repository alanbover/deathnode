package mesosMock

import (
	"fmt"
	"os"
	"encoding/json"
	"io/ioutil"
	"runtime"
	"path"
	"github.com/alanbover/deathnode/mesos"
)

type ConnectionMock struct {
	Records		map[string]*[]string
	Requests	map[string]*[]string
}

func (c* ConnectionMock) GetMesosTasks() (*mesos.TasksResponse, error) {
	mockResponse, _ := c.replay(&mesos.TasksResponse{}, "GetMesosTasks")
	return mockResponse.(*mesos.TasksResponse), nil
}

func (c* ConnectionMock) GetMesosFrameworks() (*mesos.FrameworksResponse, error) {
	mockResponse, _ := c.replay(&mesos.FrameworksResponse{}, "GetMesosFrameworks")
	return mockResponse.(*mesos.FrameworksResponse), nil
}

func (c* ConnectionMock) GetMesosSlaves() (*mesos.SlavesResponse, error) {
	mockResponse, _ := c.replay(&mesos.SlavesResponse{}, "GetMesosSlaves")
	return mockResponse.(*mesos.SlavesResponse), nil
}

func (c* ConnectionMock) SetHostInMaintenance(hostname, ip string) error {
	if c.Requests == nil {
		c.Requests = map[string]*[]string{}
	}

	c.Requests["SetHostInMaintenance"] = &[]string{hostname, ip}
	return nil
}

func (c* ConnectionMock) replay(mockResponse interface{}, templateFileName string) (interface{}, error) {

	records, ok := c.Records[templateFileName]
	if ! ok {
		fmt.Printf("AWS Mock %v method called but not defined\n", templateFileName)
		os.Exit(1)
	}

	if len(*records) == 0 {
		fmt.Printf("AWS Mock replay called more times than configured for %v\n", templateFileName)
		os.Exit(1)
	}

	currentRecord := (*records)[0]

	file, err := ioutil.ReadFile(getCurrentPath() + "/records" + "/" + currentRecord + "/" + templateFileName + ".json")
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(file, &mockResponse)
	if err != nil {
		fmt.Printf("Error loading mock json: %v\n", err)
		os.Exit(1)
	}

	*records = (*records)[1:]
	return mockResponse, nil
}

func getCurrentPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("No caller information")
	}
	return path.Dir(filename)
}
