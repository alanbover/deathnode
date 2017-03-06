package awsMock

import (
	"fmt"
	"os"
	"encoding/json"
	"io/ioutil"
	"runtime"
	"path"
)

import (
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

type ConnectionMock struct {
	Records		map[string]*[]string
	Requests	map[string]*[]string
}

func (c* ConnectionMock) DescribeInstanceById(instanceId string) (*ec2.Instance, error) {

	mockResponse, _ := c.replay(&ec2.Instance{}, "DescribeInstanceById")
	return mockResponse.(*ec2.Instance), nil
}

func (c* ConnectionMock) DescribeAGByName(autoscalingGroupName string) (*autoscaling.Group, error) {

	mockResponse, _ := c.replay(&autoscaling.Group{}, "DescribeAGByName")
	return mockResponse.(*autoscaling.Group), nil
}

func (c* ConnectionMock) DetachInstance(autoscalingGroupName, instanceId string) error {

	if c.Requests == nil {
		c.Requests = map[string]*[]string{}
	}

	c.Requests["DetachInstance"] = &[]string{autoscalingGroupName, instanceId}
	return nil
}

func (c* ConnectionMock) TerminateInstance(instanceId string) error {

	if c.Requests == nil {
		c.Requests = map[string]*[]string{}
	}

	c.Requests["TerminateInstance"] = &[]string{instanceId}
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
