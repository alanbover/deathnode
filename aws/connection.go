package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type AwsConnection struct {
	ec2         *ec2.EC2
	autoscaling *autoscaling.AutoScaling
}

type AwsConnectionInterface interface {
	DescribeInstanceById(instanceId string) (*ec2.Instance, error)
	DescribeAGByName(autoscalingGroupName string) (*autoscaling.Group, error)
	DetachInstance(autoscalingGroupName string, instanceId string) error
	TerminateInstance(instanceId string) error
}

func NewConnection(accessKey, secretKey, region, iamRole, iamSession string) (*AwsConnection, error) {

	session, err := newAwsSession(&sessionParameters{
		accessKey:  accessKey,
		secretKey:  secretKey,
		region:     region,
		iamRole:    iamRole,
		iamSession: iamSession,
	})

	if err != nil {
		fmt.Print("Error trying to create AWS session. ", err)
	}

	return &AwsConnection{
		ec2:         ec2.New(session),
		autoscaling: autoscaling.New(session),
	}, nil
}

func (c *AwsConnection) DescribeAGByName(autoscalingGroupName string) (*autoscaling.Group, error) {

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

func (c *AwsConnection) DetachInstance(autoscalingGroupName, instanceId string) error {

	instanceIds := []*string{&instanceId}
	shouldDecrementDesiredCapacity := false

	detachInstancesInput := &autoscaling.DetachInstancesInput{
		AutoScalingGroupName:           &autoscalingGroupName,
		InstanceIds:                    instanceIds,
		ShouldDecrementDesiredCapacity: &shouldDecrementDesiredCapacity,
	}

	c.autoscaling.DetachInstances(detachInstancesInput)

	return nil
}

func (c *AwsConnection) DescribeInstanceById(instanceId string) (*ec2.Instance, error) {

	filter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-id"),
				Values: aws.StringSlice([]string{instanceId}),
			},
		},
	}

	response, err := c.ec2.DescribeInstances(filter)

	if err != nil {
		return nil, err
	}

	return response.Reservations[0].Instances[0], nil
}

func (c *AwsConnection) TerminateInstance(instanceId string) error {

	instanceIds := []*string{&instanceId}

	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := c.ec2.TerminateInstances(terminateInstancesInput)

	return err
}
