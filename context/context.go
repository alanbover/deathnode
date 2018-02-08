package context

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	"github.com/benbjohnson/clock"
)

// ApplicationConf stores the application configurations
type ApplicationConf struct {
	ConstraintsType          arrayFlags
	RecommenderType          string
	DeathNodeMark            string
	AutoscalingGroupPrefixes arrayFlags
	ProtectedFrameworks      arrayFlags
	ProtectedTasksLabels     arrayFlags
	DelayDeleteSeconds       int
	ResetLifecycle           bool
}

// ApplicationContext stores the application configurations and both AWS and Mesos connections
type ApplicationContext struct {
	Conf      ApplicationConf
	AwsConn   aws.ClientInterface
	MesosConn mesos.ClientInterface
	Clock     clock.Clock
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
