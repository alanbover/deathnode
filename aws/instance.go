package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

// DeathNodeTagMark holds the value for the TAG to mark instances to be deleted
const DeathNodeTagMark = "DEATH_NODE_MARK"

type instance struct {
	autoscalingGroupID  string
	launchConfiguration string
	ipAddress           string
	instanceID          string
	markedToBeRemoved   bool
}

// InstanceMonitor monitors an AWS instance
type InstanceMonitor struct {
	instance      *instance
	awsConnection ClientInterface
}

func newInstanceMonitor(conn ClientInterface, autoscalingGroupID, instanceID string) (*InstanceMonitor, error) {

	response, err := conn.DescribeInstanceByID(instanceID)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	return &InstanceMonitor{
		instance: &instance{
			autoscalingGroupID:  autoscalingGroupID,
			ipAddress:           *response.PrivateIpAddress,
			instanceID:          instanceID,
			markedToBeRemoved:   isMarkedToBeRemoved(response.Tags),
		},
		awsConnection: conn,
	}, nil
}

// GetIP returns the private IP of the AWS instance
func (a *InstanceMonitor) GetIP() string {
	return a.instance.ipAddress
}

// GetInstanceID returns the instanceId of the instance being monitored
func (a *InstanceMonitor) GetInstanceID() string {
	return a.instance.instanceID
}

// MarkToBeRemoved sets a tag for the instance with:
// Key: valueOf(DEATH_NODE_TAG_MARK)
// Value: Current timestamp (epoch)
func (a *InstanceMonitor) MarkToBeRemoved() error {
	err := a.awsConnection.SetInstanceTag(DeathNodeTagMark, getEpochAsString(), a.instance.instanceID)
	a.instance.markedToBeRemoved = true
	return err
}

func getEpochAsString() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}

func isMarkedToBeRemoved(tags []*ec2.Tag) bool {
	for _, tag := range tags {
		if DeathNodeTagMark == *tag.Key {
			return true
		}
	}
	return false
}
