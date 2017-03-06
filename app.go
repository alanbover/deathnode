package main

import "time"
import "flag"

import (
	"github.com/alanbover/deathnode/watchers"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/aws"
)

const POLLING_MILLISECONDS = 60000

func main() {

	var accessKey = flag.String("accessKey", "", "help message for flagname")
	var secretKey = flag.String("secretKey", "", "help message for flagname")
	var region = flag.String("region", "", "help message for flagname")
	var iamRole = flag.String("iamRole", "", "help message for flagname")
	var iamSession = flag.String("iamSession", "", "help message for flagname")

	// Those should be an array
	var autoscalingGroupName = flag.String("autoscalingGroupName", "", "The autoscaling group name to monitor")
	var registeredFramework = flag.String("registeredFramework", "", "The mesos frameworks to wait for kill the node")

	aws_conn, _ := aws.NewConnection(*accessKey, *secretKey, *region, *iamRole, *iamSession)
	autoscalingGroupNames := []string{*autoscalingGroupName}
	registeredFrameworks := []string{*registeredFramework}

	mesosAutoscalingGroups, _ := mesos.NewAutoscalingGroups(aws_conn, autoscalingGroupNames)
	mesosAutoscalingWatcher := watchers.NewMesosAutoscalingWatcher(registeredFrameworks)

	ticker := time.NewTicker(time.Millisecond * POLLING_MILLISECONDS)
	for {
		for _, autoscalingGroup := range *mesosAutoscalingGroups {
			go mesosAutoscalingWatcher.CheckIfInstancesToKill(autoscalingGroup)
		}
		<-ticker.C
	}
}
