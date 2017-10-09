package main

import "time"
import "flag"

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	log "github.com/sirupsen/logrus"
)

type arrayFlags []string

var accessKey, secretKey, region, iamRole, iamSession, mesosURL, constraintsType, recommenderType, deathNodeMark string
var autoscalingGroupPrefixes, protectedFrameworks, protectedTasksLabels arrayFlags
var pollingSeconds, delayDeleteSeconds int
var debug bool

func main() {

	initFlags()
	enforceFlags()

	log.SetLevel(log.InfoLevel)
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Create the monitors for autoscaling groups
	awsConn, err := aws.NewClient(accessKey, secretKey, region, iamRole, iamSession)
	if err != nil {
		log.Fatal("Error connecting to AWS: ", err)
	}
	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, autoscalingGroupPrefixes, deathNodeMark)

	// Create the Mesos monitor
	mesosConn := &mesos.Client{
		MasterURL: mesosURL,
	}
	mesosMonitor := monitor.NewMesosMonitor(mesosConn, protectedFrameworks, protectedTasksLabels)

	// Create deathnoteWatcher
	notebook := deathnode.NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds, deathNodeMark)
	deathNodeWatcher := deathnode.NewWatcher(notebook, mesosMonitor, constraintsType, recommenderType)

	ticker := time.NewTicker(time.Second * time.Duration(pollingSeconds))
	for {
		go run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
		<-ticker.C
	}
}

func run(mesosMonitor *monitor.MesosMonitor, autoscalingGroups *monitor.AutoscalingGroupsMonitor,
	deathNodeWatcher *deathnode.Watcher) {

	log.Debug("New check triggered")
	// Refresh autoscaling monitors and mesos monitor
	autoscalingGroups.Refresh()
	mesosMonitor.Refresh()

	// For each autoscaling monitor, check if any instances needs to be removed
	for _, autoscalingGroup := range autoscalingGroups.GetAllMonitors() {
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
	flag.StringVar(&mesosURL, "mesosUrl", "", "The URL for Mesos master")

	flag.Var(&autoscalingGroupPrefixes, "autoscalingGroupName", "An autoscalingGroup prefix for monitor")
	flag.Var(&protectedFrameworks, "protectedFrameworks", "The mesos frameworks to wait for kill the node")
	flag.Var(&protectedTasksLabels, "protectedTaskLabels", "The Tasks protected with labels(boolean) to wait for kill the node")
	// Move constraints to array, so we apply multiple
	flag.StringVar(&constraintsType, "constraintsType", "noContraint", "The constrainst implementation to use")
	flag.StringVar(&recommenderType, "recommenderType", "firstAvailableAgent", "The recommender implementation to use")

	flag.StringVar(&deathNodeMark, "deathNodeMark", "DEATH_NODE_MARK", "The tag to apply for instances to be deleted")

	flag.IntVar(&pollingSeconds, "polling", 60, "Seconds between executions")
	flag.IntVar(&delayDeleteSeconds, "delayDelete", 0, "Time to wait between kill executions (in seconds)")

	flag.Parse()
}

func enforceFlags() {

	if mesosURL == "" {
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

func (i *arrayFlags) String() string {
	return ""
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}
