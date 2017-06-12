package main

import "time"
import "flag"

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
)

type ArrayFlags []string

var accessKey, secretKey, region, iamRole, iamSession, mesosUrl, constraintsType, recommenderType string
var autoscalingGroupPrefixes, protectedFrameworks ArrayFlags
var polling_seconds int
var debug bool

func main() {

	initFlags()
	enforceFlags()

	log.SetLevel(log.InfoLevel)
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Create the monitors for autoscaling groups
	awsConn, err := aws.NewConnection(accessKey, secretKey, region, iamRole, iamSession)
	if err != nil {
		log.Fatal("Error connecting to AWS: ", err)
	}
	autoscalingGroups, _ := aws.NewAutoscalingGroups(awsConn, autoscalingGroupPrefixes)

	// Create the Mesos monitor
	mesosConn := &mesos.MesosConnection{
		MasterUrl: mesosUrl,
	}
	mesosMonitor := mesos.NewMesosMonitor(mesosConn, protectedFrameworks)

	// Create deathnoteWatcher
	notebook := deathnode.NewNotebook(autoscalingGroups, awsConn, mesosMonitor)
	deathNodeWatcher := deathnode.NewDeathNodeWatcher(notebook, mesosMonitor, constraintsType, recommenderType)

	ticker := time.NewTicker(time.Second * time.Duration(polling_seconds))
	for {
		go Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
		<-ticker.C
	}
}

func Run(mesosMonitor *mesos.MesosMonitor, autoscalingGroups *aws.AutoscalingGroups,
	deathNodeWatcher *deathnode.DeathNodeWatcher) {

	log.Debug("New check triggered")
	// Refresh autoscaling monitors and mesos monitor
	autoscalingGroups.Refresh()
	mesosMonitor.Refresh()

	// For each autoscaling monitor, check if any instances needs to be removed
	for _, autoscalingGroup := range autoscalingGroups.GetMonitors() {
		deathNodeWatcher.RemoveUndesiredInstances(autoscalingGroup)
	}

	// Check if any agents are drained, so we can remove them from AWS
	deathNodeWatcher.DestroyInstancesAttempt()
}

func initFlags() {

	flag.StringVar(&accessKey, "accessKey", "", "help message for flagname")
	flag.StringVar(&secretKey, "secretKey", "", "help message for flagname")
	flag.StringVar(&region, "region", "eu-west-1", "help message for flagname")
	flag.StringVar(&iamRole, "iamRole", "", "help message for flagname")
	flag.StringVar(&iamSession, "iamSession", "", "help message for flagname")

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&mesosUrl, "mesosUrl", "", "The URL for Mesos master")

	flag.Var(&autoscalingGroupPrefixes, "autoscalingGroupName", "An autoscalingGroup prefix for monitor")
	flag.Var(&protectedFrameworks, "protectedFrameworks", "The mesos frameworks to wait for kill the node")

	// Move constraints to array, so we apply multiple
	flag.StringVar(&constraintsType, "constraintsType", "noContraint", "The constrainst implementation to use")
	flag.StringVar(&recommenderType, "recommenderType", "firstAvailableAgent", "The recommender implementation to use")

	flag.IntVar(&polling_seconds, "polling", 60, "Seconds between executions")

	flag.Parse()
}

func enforceFlags() {

	if mesosUrl == "" {
		flag.Usage()
		log.Fatal("mesosUrl flag is required")
	}

	if len(autoscalingGroupPrefixes) < 1 {
		flag.Usage()
		log.Fatal("at least one autoscalingGroupName flag is required")
	}

	if len(protectedFrameworks) < 1 {
		flag.Usage()
		log.Fatal("at least one registeredFramework flag is required")
	}
}

func (i *ArrayFlags) String() string {
	return ""
}

func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
