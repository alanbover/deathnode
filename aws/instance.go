package aws

type Instance struct {
	autoscalingGroupId  string
	launchConfiguration string
	ipAddress           string
	instanceId          string
}

type InstanceMonitor struct {
	instance      *Instance
	awsConnection AwsConnectionInterface
}

func NewInstanceMonitor(conn AwsConnectionInterface, autoscalingGroupId, launchConfiguration, instanceId string) (*InstanceMonitor, error) {

	response, err := conn.DescribeInstanceById(instanceId)

	if err != nil {
		return &InstanceMonitor{}, err
	}

	return &InstanceMonitor{
		instance: &Instance{
			autoscalingGroupId:  autoscalingGroupId,
			launchConfiguration: launchConfiguration,
			ipAddress:           *response.PrivateIpAddress,
			instanceId:          instanceId,
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
