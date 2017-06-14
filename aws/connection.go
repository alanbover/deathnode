package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strings"
)

type AwsConnection struct {
	ec2         *ec2.EC2
	autoscaling *autoscaling.AutoScaling
}

type AwsConnectionInterface interface {
	DescribeInstanceById(instanceId string) (*ec2.Instance, error)
	DescribeInstancesByTag(tagKey string) ([]*ec2.Instance, error)
	DescribeAGByName(autoscalingGroupName string) ([]*autoscaling.Group, error)
	DetachInstance(autoscalingGroupName string, instanceId string) error
	TerminateInstance(instanceId string) error
	SetASGInstanceProtection(autoscalingGroupName *string, instanceIds []*string) error
	SetInstanceTag(key, value, instanceId string) error
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

func (c *AwsConnection) DescribeAGByName(autoscalingGroupPrefix string) ([]*autoscaling.Group, error) {

	autoscalingGroupList := []*autoscaling.Group{}

	filter := &autoscaling.DescribeAutoScalingGroupsInput{}
	response, err := c.autoscaling.DescribeAutoScalingGroups(filter)
	if err != nil {
		return nil, err
	}

	autoscalingGroupList = appendASGByPrefix(autoscalingGroupList, response.AutoScalingGroups, autoscalingGroupPrefix)
	for response.NextToken != nil {
		nextToken := response.NextToken
		response, err = c.describeAGByNameWithToken(nextToken)
		if err != nil {
			return nil, err
		}

		autoscalingGroupList = appendASGByPrefix(autoscalingGroupList, response.AutoScalingGroups, autoscalingGroupPrefix)
	}

	return autoscalingGroupList, nil
}

func (c *AwsConnection) describeAGByNameWithToken(nextToken *string) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {

	filter := &autoscaling.DescribeAutoScalingGroupsInput{
		NextToken: nextToken,
	}

	response, err := c.autoscaling.DescribeAutoScalingGroups(filter)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func appendASGByPrefix(asgResponse, asgToFilter []*autoscaling.Group, prefix string) []*autoscaling.Group {

	for _, autoscalingGroup := range asgToFilter {
		if strings.HasPrefix(*autoscalingGroup.AutoScalingGroupName, prefix) {
			asgResponse = append(asgResponse, autoscalingGroup)
		}
	}

	return asgResponse
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
		Filters: []*ec2.Filter{{
			Name:   aws.String("instance-id"),
			Values: aws.StringSlice([]string{instanceId}),
		}},
	}

	response, err := c.ec2.DescribeInstances(filter)

	if err != nil {
		return nil, err
	}

	return response.Reservations[0].Instances[0], nil
}

func (c *AwsConnection) DescribeInstancesByTag(tagKey string) ([]*ec2.Instance, error) {

	instances := []*ec2.Instance{}

	filter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("tag-key"),
				Values: aws.StringSlice([]string{tagKey}),
			},
			{
				Name:   aws.String("instance-state-name"),
				Values: aws.StringSlice([]string{"running"}),
			},
		},
	}

	response, err := c.ec2.DescribeInstances(filter)

	if err != nil {
		return nil, err
	}

	if len(response.Reservations) == 0 {
		return []*ec2.Instance{}, nil
	}

	for _, reservation := range response.Reservations {
		for _, instance := range reservation.Instances {
			instances = append(instances, instance)
		}
	}

	return instances, nil
}

func (c *AwsConnection) TerminateInstance(instanceId string) error {

	instanceIds := []*string{&instanceId}

	terminateInstancesInput := &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	}

	_, err := c.ec2.TerminateInstances(terminateInstancesInput)

	return err
}

func (c *AwsConnection) SetASGInstanceProtection(autoscalingGroupName *string, instanceIds []*string) error {

	instancesProtectedFromScaleIn := true
	updateAutoScalingGroupInput := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName:             autoscalingGroupName,
		NewInstancesProtectedFromScaleIn: &instancesProtectedFromScaleIn,
	}

	_, err := c.autoscaling.UpdateAutoScalingGroup(updateAutoScalingGroupInput)

	if err != nil {
		return err
	}

	protectedFromScaleIn := true
	setInstanceProtectionInput := &autoscaling.SetInstanceProtectionInput{
		AutoScalingGroupName: autoscalingGroupName,
		InstanceIds:          instanceIds,
		ProtectedFromScaleIn: &protectedFromScaleIn,
	}

	_, err = c.autoscaling.SetInstanceProtection(setInstanceProtectionInput)

	return err
}

func (c *AwsConnection) SetInstanceTag(key, value, instanceId string) error {

	tag := &ec2.Tag{
		Key:   aws.String(key),
		Value: aws.String(value),
	}

	_, err := c.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceId)},
		Tags:      []*ec2.Tag{tag},
	})

	return err
}
