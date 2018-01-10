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

// FlushMock will flush all requests recorded on the mock side
func (c *ConnectionMock) FlushMock() {
	c.Requests = map[string][][]string{}
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

// DescribeAGsByPrefix is a mock call for testing purposes
func (c *ConnectionMock) DescribeAGsByPrefix(autoscalingGroupName string) ([]*autoscaling.Group, error) {

	mockResponse, _ := c.replay(&[]*autoscaling.Group{}, "DescribeAGByName")
	return *mockResponse.(*[]*autoscaling.Group), nil
}

// SetASGInstanceProtection is a mock call for testing purposes
func (c *ConnectionMock) SetASGInstanceProtection(autoscalingGroupName *string, instanceIDs []*string) error {

	inputValues := []string{*autoscalingGroupName}
	for _, instanceID := range instanceIDs {
		inputValues = append(inputValues, *instanceID)
	}

	c.addRequests("SetASGInstanceProtection", inputValues)
	return nil
}

// RemoveASGInstanceProtection is a mock call for testing purposes
func (c *ConnectionMock) RemoveASGInstanceProtection(autoscalingGroupName, instanceID *string) error {

	c.addRequests("RemoveASGInstanceProtection", []string{*autoscalingGroupName, *instanceID})
	return nil
}

// SetInstanceTag is a mock call for testing purposes
func (c *ConnectionMock) SetInstanceTag(key, value, instanceID string) error {

	inputValues := []string{key, value, instanceID}
	c.addRequests("SetInstanceTag", inputValues)
	return nil
}

// HasLifeCycleHook is a mock call for testing purposes
func (c *ConnectionMock) HasLifeCycleHook(autoscalingGroupName string) (bool, error) {

	records, ok := c.Records["HasLifeCycleHook"]
	if !ok {
		return false, nil
	}

	hasLifeCycleHook := (*records)[0]
	*records = (*records)[1:]
	return hasLifeCycleHook == "true", nil
}

// PutLifeCycleHook is a mock call for testing purposes
func (c *ConnectionMock) PutLifeCycleHook(autoscalingGroupName string, heartbeatTimeout *int64) error {

	c.addRequests("PutLifeCycleHook", []string{autoscalingGroupName, fmt.Sprintf("%d", *heartbeatTimeout)})
	return nil
}

// CompleteLifecycleAction is a mock call for testing purposes
func (c *ConnectionMock) CompleteLifecycleAction(autoscalingGroupName, instanceID *string) error {

	c.addRequests("CompleteLifecycleAction", []string{*autoscalingGroupName, *instanceID})
	return nil
}

// RecordLifecycleActionHeartbeat is a mock call for testing purposes
func (c *ConnectionMock) RecordLifecycleActionHeartbeat(autoscalingGroupName, instanceID *string) error {

	c.addRequests("RecordLifecycleActionHeartbeat", []string{*autoscalingGroupName, *instanceID})
	return nil
}

func (c *ConnectionMock) addRequests(funcName string, parameters []string) {

	if c.Requests == nil {
		c.Requests = map[string][][]string{}
	}

	c.Requests[funcName] = append(c.Requests[funcName], parameters)
}

func (c *ConnectionMock) replay(mockResponse interface{}, templateFileName string) (interface{}, error) {

	records := c.getRecords(templateFileName)
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

func (c *ConnectionMock) getRecords(templateFileName string) *[]string {

	records, ok := c.Records[templateFileName]
	if !ok {
		fmt.Printf("AWS Mock %v method called but not defined\n", templateFileName)
		os.Exit(1)
	}

	if len(*records) == 0 {
		fmt.Printf("AWS Mock replay called more times than configured for %v\n", templateFileName)
		os.Exit(1)
	}
	return records
}
