package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
)

func (c* Connection) DescribeInstanceById(instanceId string) (*ec2.Instance, error) {

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

func (c* Connection) TerminateInstance(instanceId string) error {

	instanceIds := []*string{&instanceId}

	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := c.ec2.TerminateInstances(terminateInstancesInput)

	return err
}
