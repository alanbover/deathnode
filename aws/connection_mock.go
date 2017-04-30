package aws

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"io/ioutil"
	"os"
	"path/filepath"
)

type AwsConnectionMock struct {
	Records  map[string]*[]string
	Requests map[string][][]string
}

func (c *AwsConnectionMock) DescribeInstanceById(instanceId string) (*ec2.Instance, error) {

	mockResponse, _ := c.replay(&ec2.Instance{}, "DescribeInstanceById")
	return mockResponse.(*ec2.Instance), nil
}

func (c *AwsConnectionMock) DescribeAGByName(autoscalingGroupName string) ([]*autoscaling.Group, error) {

	mockResponse, _ := c.replay(&[]*autoscaling.Group{}, "DescribeAGByName")
	return *mockResponse.(*[]*autoscaling.Group), nil
}

func (c *AwsConnectionMock) DetachInstance(autoscalingGroupName, instanceId string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	c.Requests["DetachInstance"] = append(c.Requests["DetachInstance"], []string{autoscalingGroupName, instanceId})
	return nil
}

func (c *AwsConnectionMock) TerminateInstance(instanceId string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	c.Requests["TerminateInstance"] = append(c.Requests["TerminateInstance"], []string{instanceId})
	return nil
}

func (c *AwsConnectionMock) SetASGInstanceProtection(autoscalingGroupName *string, instanceIds []*string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	input_values := []string{*autoscalingGroupName}
	for _, instanceId := range instanceIds {
		input_values = append(input_values, *instanceId)
	}

	c.Requests["SetASGInstanceProtection"] = append(c.Requests["SetASGInstanceProtection"], input_values)
	return nil
}

func (c *AwsConnectionMock) replay(mockResponse interface{}, templateFileName string) (interface{}, error) {

	records, ok := c.Records[templateFileName]
	if !ok {
		fmt.Printf("AWS Mock %v method called but not defined\n", templateFileName)
		os.Exit(1)
	}

	if len(*records) == 0 {
		fmt.Printf("AWS Mock replay called more times than configured for %v\n", templateFileName)
		os.Exit(1)
	}

	currentRecord := (*records)[0]

	file, err := ioutil.ReadFile(getCurrentPath() + "/testdata" + "/" + currentRecord + "/" + templateFileName + ".json")
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

	gopath := os.Getenv("GOPATH")
	return filepath.Join(gopath, "src/github.com/alanbover/deathnode/aws")
}
