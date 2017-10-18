// +build !test

package aws

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strings"
)

const (
	lifecycleHookName                   = "DEATHNODE"
	continueString                      = "CONTINUE"
	lifecycleTransitionTerminationState = "autoscaling:EC2_INSTANCE_TERMINATING"
)

// Client holds the AWS SDK objects for call AWS API
type Client struct {
	ec2         *ec2.EC2
	autoscaling *autoscaling.AutoScaling
}

// ClientInterface implements a client with all required operations against AWS API
type ClientInterface interface {
	DescribeInstanceByID(instanceID string) (*ec2.Instance, error)
	DescribeInstancesByTag(tagKey string) ([]*ec2.Instance, error)
	DescribeAGsByPrefix(autoscalingGroupName string) ([]*autoscaling.Group, error)
	RemoveASGInstanceProtection(autoscalingGroupName, instanceID *string) error
	SetASGInstanceProtection(autoscalingGroupName *string, instanceIDs []*string) error
	SetInstanceTag(key, value, instanceID string) error
	HasLifeCycleHook(autoscalingGroupName string) (bool, error)
	PutLifeCycleHook(autoscalingGroupName string, heartbeatTimeout *int64) error
	CompleteLifecycleAction(autoscalingGroupName, instanceID *string) error
}

// NewClient returns a new aws.client
func NewClient(accessKey, secretKey, region, iamRole, iamSession string) (*Client, error) {

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

	return &Client{
		ec2:         ec2.New(session),
		autoscaling: autoscaling.New(session),
	}, nil
}

// CompleteLifecycleAction completes a lifecycle event for an instance pending to be deleted
func (c *Client) CompleteLifecycleAction(autoscalingGroupName, instanceID *string) error {

	completeLifecycleActionInput := &autoscaling.CompleteLifecycleActionInput{
		AutoScalingGroupName:  autoscalingGroupName,
		InstanceId:            instanceID,
		LifecycleActionResult: aws.String(continueString),
		LifecycleHookName:     aws.String(lifecycleHookName),
	}

	_, err := c.autoscaling.CompleteLifecycleAction(completeLifecycleActionInput)
	return err
}

// HasLifeCycleHook checks if deathnode lifecyclehook is enabled for an autoscalingGroup
func (c *Client) HasLifeCycleHook(autoscalingGroupName string) (bool, error) {

	describeLifecycleHooksInput := &autoscaling.DescribeLifecycleHooksInput{
		AutoScalingGroupName: aws.String(autoscalingGroupName),
		LifecycleHookNames:   []*string{aws.String(lifecycleHookName)},
	}

	describeLifecycleHooksOutput, err := c.autoscaling.DescribeLifecycleHooks(describeLifecycleHooksInput)
	if err != nil {
		return false, err
	}

	return len(describeLifecycleHooksOutput.LifecycleHooks) != 0, nil
}

// PutLifeCycleHook adds an INSTANCE_TERMINATING lifecycle hook to an autoscalingGroup
func (c *Client) PutLifeCycleHook(autoscalingGroupName string, heartbeatTimeout *int64) error {

	putLifecycleHookInput := &autoscaling.PutLifecycleHookInput{
		AutoScalingGroupName: aws.String(autoscalingGroupName),
		DefaultResult:        aws.String(continueString),
		HeartbeatTimeout:     heartbeatTimeout,
		LifecycleHookName:    aws.String(lifecycleHookName),
		LifecycleTransition:  aws.String(lifecycleTransitionTerminationState),
	}

	_, err := c.autoscaling.PutLifecycleHook(putLifecycleHookInput)
	return err
}

// DescribeAGsByPrefix returns all autoscaling groups that matches a certain prefix
func (c *Client) DescribeAGsByPrefix(autoscalingGroupPrefix string) ([]*autoscaling.Group, error) {

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

func (c *Client) describeAGByNameWithToken(nextToken *string) (*autoscaling.DescribeAutoScalingGroupsOutput, error) {

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

// DescribeInstanceByID returns the instance that matches an instanceID
func (c *Client) DescribeInstanceByID(instanceID string) (*ec2.Instance, error) {

	filter := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{{
			Name:   aws.String("instance-id"),
			Values: aws.StringSlice([]string{instanceID}),
		}},
	}

	response, err := c.ec2.DescribeInstances(filter)

	if err != nil {
		return nil, err
	}

	if len(response.Reservations) < 1 || len(response.Reservations[0].Instances) < 1 {
		return nil, fmt.Errorf("No instance information found for instance id %v", instanceID)
	}

	return response.Reservations[0].Instances[0], nil
}

// DescribeInstancesByTag return all instances with a certain tag set
func (c *Client) DescribeInstancesByTag(tagKey string) ([]*ec2.Instance, error) {

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

// RemoveASGInstanceProtection remove ProtectFromScaleIn flag from some instances from an autoscalingGroup
func (c *Client) RemoveASGInstanceProtection(autoscalingGroupName, instanceID *string) error {

	protectedFromScaleIn := false
	setInstanceProtectionInput := &autoscaling.SetInstanceProtectionInput{
		AutoScalingGroupName: autoscalingGroupName,
		InstanceIds:          []*string{instanceID},
		ProtectedFromScaleIn: &protectedFromScaleIn,
	}

	_, err := c.autoscaling.SetInstanceProtection(setInstanceProtectionInput)

	return err
}

// SetASGInstanceProtection set an autoscalingGroup and all it's instances with ProtectFromScaleIn flag
func (c *Client) SetASGInstanceProtection(autoscalingGroupName *string, instanceIds []*string) error {

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

// SetInstanceTag set a tag with <key,value> to an AWS instance
func (c *Client) SetInstanceTag(key, value, instanceID string) error {

	tag := &ec2.Tag{
		Key:   aws.String(key),
		Value: aws.String(value),
	}

	_, err := c.ec2.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{aws.String(instanceID)},
		Tags:      []*ec2.Tag{tag},
	})

	return err
}
