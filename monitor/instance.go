package monitor

import (
	"fmt"
	"github.com/alanbover/deathnode/context"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
	"strconv"
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
	tagRemovalTimestamp int64
	ctx                 *context.ApplicationContext
}

func newInstanceMonitor(ctx *context.ApplicationContext, autoscalingGroupID, instanceID, lifecycleState string,
	isProtected bool) (*InstanceMonitor, error) {

	response, err := ctx.AwsConn.DescribeInstanceByID(instanceID)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	tagRemovalTimestamp, err := getTagRemovalTimestamp(response.Tags, ctx.Conf.DeathNodeMark)
	if err != nil {
		log.Warn("Invalid value found for tag %s on instance %s", ctx.Conf.DeathNodeMark, instanceID)
	}

	return &InstanceMonitor{
		autoscalingGroupID:  autoscalingGroupID,
		ipAddress:           *response.PrivateIpAddress,
		instanceID:          instanceID,
		lifecycleState:      lifecycleState,
		isProtected:         isProtected,
		ctx:                 ctx,
		tagRemovalTimestamp: tagRemovalTimestamp,
	}, nil
}

// IP returns the private IP of the AWS instance
func (a *InstanceMonitor) IP() string {
	return a.ipAddress
}

// TagRemovalTimestamp returns the start timestamp for the lifecycle hook
func (a *InstanceMonitor) TagRemovalTimestamp() int64 {
	return a.tagRemovalTimestamp
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
	currentTimestamp := a.ctx.Clock.Now().Unix()
	err := a.ctx.AwsConn.SetInstanceTag(a.ctx.Conf.DeathNodeMark, fmt.Sprintf("%v", currentTimestamp), a.instanceID)
	a.tagRemovalTimestamp = currentTimestamp
	return err
}

// IsMarkedToBeRemoved is true when the instance has been marked for removal
func (a *InstanceMonitor) IsMarkedToBeRemoved() bool {
	return a.tagRemovalTimestamp != 0
}

func (a *InstanceMonitor) setLifecycleState(lifecycleState string) {
	a.lifecycleState = lifecycleState

	if lifecycleState == LifecycleStateTerminatingWait && a.isProtected {
		// A non-controled instance went to Terminating:Wait, probably because it went unhealthy
		a.TagToBeRemoved()
	}
}

func getTagRemovalTimestamp(tags []*ec2.Tag, deathNodeMark string) (int64, error) {
	for _, tag := range tags {
		if deathNodeMark == *tag.Key {
			timestamp, err := strconv.ParseInt(*tag.Value, 10, 64)
			if err != nil {
				return 0, err
			}
			return timestamp, nil
		}
	}
	return 0, nil
}
