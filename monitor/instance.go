package monitor

import (
	"fmt"
	"github.com/alanbover/deathnode/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

// LifecycleStateTerminatingWait defines the state of an instance in the autoscalingGroup when it's waiting for
// confirmation to be removed
const LifecycleStateTerminatingWait = "Terminating:Wait"

type instance struct {
	autoscalingGroupID  string
	launchConfiguration string
	ipAddress           string
	instanceID          string
	lifecycleState      string
	isProtected         bool
	isMarkedToBeRemoved bool
}

// InstanceMonitor monitors an AWS instance
type InstanceMonitor struct {
	instance      *instance
	awsConnection aws.ClientInterface
	deathNodeMark string
}

func newInstanceMonitor(conn aws.ClientInterface, autoscalingGroupID, instanceID, deathNodeMark, lifecycleState string, isProtected bool) (*InstanceMonitor, error) {

	response, err := conn.DescribeInstanceByID(instanceID)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	return &InstanceMonitor{
		instance: &instance{
			autoscalingGroupID:  autoscalingGroupID,
			ipAddress:           *response.PrivateIpAddress,
			instanceID:          instanceID,
			isMarkedToBeRemoved: isMarkedToBeRemoved(response.Tags, deathNodeMark),
			lifecycleState:      lifecycleState,
			isProtected:         isProtected,
		},
		awsConnection: conn,
		deathNodeMark: deathNodeMark,
	}, nil
}

// GetIP returns the private IP of the AWS instance
func (a *InstanceMonitor) GetIP() string {
	return a.instance.ipAddress
}

// GetLifecycleState returns the lifeCycleState of the instance in the ASG
func (a *InstanceMonitor) GetLifecycleState() string {
	return a.instance.lifecycleState
}

// GetInstanceID returns the instanceId of the instance being monitored
func (a *InstanceMonitor) GetInstanceID() *string {
	return &a.instance.instanceID
}

// GetAutoscalingGroupID returns the AutoscalingGroupId of the instance being monitored
func (a *InstanceMonitor) GetAutoscalingGroupID() *string {
	return &a.instance.autoscalingGroupID
}

// RemoveInstanceProtection removes the instance protection for the autoscaling
func (a *InstanceMonitor) RemoveInstanceProtection() error {
	err := a.awsConnection.RemoveASGInstanceProtection(&a.instance.autoscalingGroupID, []*string{&a.instance.instanceID})
	if err != nil {
		return err
	}
	a.instance.isProtected = false
	return nil
}

// IsProtected returns true if the instance has the flag instanceProtection in the ASG
func (a *InstanceMonitor) IsProtected() bool {
	return a.instance.isProtected
}

// MarkToBeRemoved sets a tag for the instance with:
// Key: valueOf(DEATH_NODE_TAG_MARK)
// Value: Current timestamp (epoch)
func (a *InstanceMonitor) MarkToBeRemoved() error {
	err := a.awsConnection.SetInstanceTag(a.deathNodeMark, getEpochAsString(), a.instance.instanceID)
	a.instance.isMarkedToBeRemoved = true
	return err
}

func (a *InstanceMonitor) setLifecycleState(lifecycleState string) {
	a.instance.lifecycleState = lifecycleState

	if lifecycleState == LifecycleStateTerminatingWait && a.instance.isProtected {
		// A non-controled instance went to Terminating:Wait, probably because it went unhealthy
		a.MarkToBeRemoved()
	}
}

func getEpochAsString() string {
	return fmt.Sprintf("%v", time.Now().Unix())
}

func isMarkedToBeRemoved(tags []*ec2.Tag, deathNodeMark string) bool {
	for _, tag := range tags {
		if deathNodeMark == *tag.Key {
			return true
		}
	}
	return false
}
