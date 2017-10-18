package context

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
)

type ApplicationConf struct {
	ConstraintsType 		string
	RecommenderType 		string
	DeathNodeMark 			string
	AutoscalingGroupPrefixes	arrayFlags
	ProtectedFrameworks		arrayFlags
	ProtectedTasksLabels 		arrayFlags
	DelayDeleteSeconds 		int
}

type ApplicationContext struct {
	Conf      ApplicationConf
	AwsConn   aws.ClientInterface
	MesosConn mesos.ClientInterface
}

type arrayFlags []string

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
