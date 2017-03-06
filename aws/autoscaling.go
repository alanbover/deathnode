package aws

import (
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func (c* Connection) DescribeAGByName(autoscalingGroupName string) (*autoscaling.Group, error) {

	filter := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			&autoscalingGroupName,
		},
	}

	response, err := c.autoscaling.DescribeAutoScalingGroups(filter)

	if err != nil {
		return nil, err
	}

	return response.AutoScalingGroups[0], nil
}

func (c* Connection) DetachInstance(autoscalingGroupName, instanceId string) error {

	instanceIds := []*string{&instanceId}
	shouldDecrementDesiredCapacity := false

	detachInstancesInput := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName: &autoscalingGroupName,
		InstanceIds: instanceIds,
		ShouldDecrementDesiredCapacity: &shouldDecrementDesiredCapacity,
	}

	c.autoscaling.DetachInstances(detachInstancesInput)

	return nil
}
