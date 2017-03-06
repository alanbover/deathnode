package aws

import "github.com/aws/aws-sdk-go/service/autoscaling"
import "github.com/aws/aws-sdk-go/service/ec2"

type Connection struct {
	ec2     	*ec2.EC2
	autoscaling 	*autoscaling.AutoScaling
}

type ConnectionInterface interface {
	DescribeInstanceById(instanceId string) (*ec2.Instance, error)
	DescribeAGByName(autoscalingGroupName string) (*autoscaling.Group, error)
	DetachInstance(autoscalingGroupName string, instanceId string) error
	TerminateInstance(instanceId string) error
}

func NewConnection(accessKey, secretKey, region, iamRole, iamSession string) (*Connection, error) {

	session, _ := newAwsSession(&sessionParameters{
		accessKey: accessKey,
		secretKey: secretKey,
		region: region,
		iamRole: iamRole,
		iamSession: iamSession,
	})

	return &Connection{
		ec2: ec2.New(session),
		autoscaling: autoscaling.New(session),
	}, nil
}
