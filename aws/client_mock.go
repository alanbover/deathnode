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

// ConnectionMock is a aws mock client for testing purposes
type ConnectionMock struct {
	Records  map[string]*[]string
	Requests map[string][][]string
}

// DescribeInstanceByID is a mock call for testing purposes
func (c *ConnectionMock) DescribeInstanceByID(instanceID string) (*ec2.Instance, error) {

	mockResponse, _ := c.replay(&ec2.Instance{}, "DescribeInstanceById")
	return mockResponse.(*ec2.Instance), nil
}

// DescribeInstancesByTag is a mock call for testing purposes
func (c *ConnectionMock) DescribeInstancesByTag(tagKey string) ([]*ec2.Instance, error) {

	mockResponse, _ := c.replay(&[]*ec2.Instance{}, "DescribeInstancesByTag")
	return *mockResponse.(*[]*ec2.Instance), nil
}

// DescribeAGByName is a mock call for testing purposes
func (c *ConnectionMock) DescribeAGByName(autoscalingGroupName string) ([]*autoscaling.Group, error) {

	mockResponse, _ := c.replay(&[]*autoscaling.Group{}, "DescribeAGByName")
	return *mockResponse.(*[]*autoscaling.Group), nil
}

// DetachInstance is a mock call for testing purposes
func (c *ConnectionMock) DetachInstance(autoscalingGroupName, instanceID string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	c.Requests["DetachInstance"] = append(c.Requests["DetachInstance"], []string{autoscalingGroupName, instanceID})
	return nil
}

// TerminateInstance is a mock call for testing purposes
func (c *ConnectionMock) TerminateInstance(instanceID string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	c.Requests["TerminateInstance"] = append(c.Requests["TerminateInstance"], []string{instanceID})
	return nil
}

// SetASGInstanceProtection is a mock call for testing purposes
func (c *ConnectionMock) SetASGInstanceProtection(autoscalingGroupName *string, instanceIDs []*string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	inputValues := []string{*autoscalingGroupName}
	for _, instanceID := range instanceIDs {
		inputValues = append(inputValues, *instanceID)
	}

	c.Requests["SetASGInstanceProtection"] = append(c.Requests["SetASGInstanceProtection"], inputValues)
	return nil
}

// SetInstanceTag is a mock call for testing purposes
func (c *ConnectionMock) SetInstanceTag(key, value, instanceID string) error {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	inputValues := []string{key, value, instanceID}

	c.Requests["SetInstanceTag"] = append(c.Requests["SetInstanceTag"], inputValues)
	return nil
}

func (c *ConnectionMock) replay(mockResponse interface{}, templateFileName string) (interface{}, error) {

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

	fileContent, err := ioutil.ReadFile(getCurrentPath() + "/testdata" + "/" + currentRecord + "/" + templateFileName + ".json")
	if err != nil {
		fmt.Printf("File error: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(fileContent, &mockResponse)
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
