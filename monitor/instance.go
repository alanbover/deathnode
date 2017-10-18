package monitor

import (
	"fmt"
	"github.com/alanbover/deathnode/context"
	"github.com/aws/aws-sdk-go/service/ec2"
	"time"
)

// LifecycleStateTerminatingWait defines the state of an instance in the autoscalingGroup when it's waiting for
// confirmation to be removed
const LifecycleStateTerminatingWait = "Terminating:Wait"

// InstanceMonitor monitors an AWS instance
type InstanceMonitor struct {
	autoscalingGroupID  string
	launchConfiguration string
	ipAddress           string
	instanceID          string
	lifecycleState      string
	isProtected         bool
	isTagToBeRemoved    bool
	ctx                 *context.ApplicationContext
}

func newInstanceMonitor(ctx *context.ApplicationContext, autoscalingGroupID, instanceID, lifecycleState string,
	isProtected bool) (*InstanceMonitor, error) {

	response, err := ctx.AwsConn.DescribeInstanceByID(instanceID)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	return &InstanceMonitor{
		autoscalingGroupID: autoscalingGroupID,
		ipAddress:          *response.PrivateIpAddress,
		instanceID:         instanceID,
		isTagToBeRemoved:   isMarkedToBeRemoved(response.Tags, ctx.Conf.DeathNodeMark),
		lifecycleState:     lifecycleState,
		isProtected:        isProtected,
		ctx:                ctx,
	}, nil
}

// IP returns the private IP of the AWS instance
func (a *InstanceMonitor) IP() string {
	return a.ipAddress
}

// LifecycleState returns the lifeCycleState of the instance in the ASG
func (a *InstanceMonitor) LifecycleState() string {
	return a.lifecycleState
}

// InstanceID returns the instanceId of the instance being monitored
func (a *InstanceMonitor) InstanceID() *string {
	return &a.instanceID
}

// AutoscalingGroupID returns the AutoscalingGroupId of the instance being monitored
func (a *InstanceMonitor) AutoscalingGroupID() *string {
	return &a.autoscalingGroupID
}

// RemoveInstanceProtection removes the instance protection for the autoscaling
func (a *InstanceMonitor) RemoveInstanceProtection() error {
	err := a.ctx.AwsConn.RemoveASGInstanceProtection(&a.autoscalingGroupID, &a.instanceID)
	if err != nil {
		return err
	}
	a.isProtected = false
	return nil
}

// IsProtected returns true if the instance has the flag instanceProtection in the ASG
func (a *InstanceMonitor) IsProtected() bool {
	return a.isProtected
}

// TagToBeRemoved sets a tag for the instance with:
// Key: valueOf(DEATH_NODE_TAG_MARK)
// Value: Current timestamp (epoch)
func (a *InstanceMonitor) TagToBeRemoved() error {
	err := a.ctx.AwsConn.SetInstanceTag(a.ctx.Conf.DeathNodeMark, epochAsString(), a.instanceID)
	a.isTagToBeRemoved = true
	return err
}

func (a *InstanceMonitor) setLifecycleState(lifecycleState string) {
	a.lifecycleState = lifecycleState

	if lifecycleState == LifecycleStateTerminatingWait && a.isProtected {
		// A non-controled instance went to Terminating:Wait, probably because it went unhealthy
		a.TagToBeRemoved()
	}
}

func epochAsString() string {
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
