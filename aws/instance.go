package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

const DEATH_NODE_TAG_MARK = "DEATH_NODE_MARK"

type Instance struct {
	autoscalingGroupId  string
	launchConfiguration string
	ipAddress           string
	instanceId          string
	markedToBeRemoved   bool
}

type InstanceMonitor struct {
	instance      *Instance
	awsConnection AwsConnectionInterface
}

func NewInstanceMonitor(conn AwsConnectionInterface, autoscalingGroupId, instanceId string) (*InstanceMonitor, error) {

	response, err := conn.DescribeInstanceById(instanceId)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	return &InstanceMonitor{
		instance: &Instance{
			autoscalingGroupId:  autoscalingGroupId,
			ipAddress:           *response.PrivateIpAddress,
			instanceId:          instanceId,
			markedToBeRemoved:   isMarkedToBeRemoved(response.Tags),
		},
		awsConnection: conn,
	}, nil
}

func (a *InstanceMonitor) Destroy() error {
	err := a.awsConnection.TerminateInstance(a.instance.instanceId)
	return err
}

func (a *InstanceMonitor) RemoveFromAutoscalingGroup() error {
	err := a.awsConnection.DetachInstance(a.instance.autoscalingGroupId, a.instance.instanceId)
	return err
}

func (a *InstanceMonitor) GetIP() string {
	return a.instance.ipAddress
}

func (a *InstanceMonitor) GetInstanceId() string {
	return a.instance.instanceId
}

func (a *InstanceMonitor) MarkToBeRemoved() error {
	err := a.awsConnection.SetInstanceTag(DEATH_NODE_TAG_MARK, getEpochAsString(), a.instance.instanceId)
	a.instance.markedToBeRemoved = true
	return err
}

func getEpochAsString() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}

func isMarkedToBeRemoved(tags []*ec2.Tag) bool {
	for _, tag := range tags {
		if DEATH_NODE_TAG_MARK == *tag.Key {
			return true
		}
	}
	return false
}
