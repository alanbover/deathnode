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
var autoscalingGroupNames, protectedFrameworks ArrayFlags
var polling_seconds int
var debug bool

func main() {

	initFlags()
	enforceFlags()

	log.SetLevel(log.InfoLevel)
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	aws_conn, err := aws.NewConnection(accessKey, secretKey, region, iamRole, iamSession)
	if err != nil {
		log.Fatal("Error connecting to AWS: ", err)
	}
	autoscalingGroups, _ := aws.NewAutoscalingGroups(aws_conn, autoscalingGroupNames)

	mesosMonitor := mesos.NewMesosMonitor(
		&mesos.MesosConnection{
			MasterUrl: mesosUrl,
		})

	notebook := deathnode.NewNotebook(mesosMonitor, protectedFrameworks)
	deathNodeWatcher := deathnode.NewDeathNodeWatcher(notebook, mesosMonitor, constraintsType, recommenderType)

	ticker := time.NewTicker(time.Second * time.Duration(polling_seconds))
	for {
		mesosMonitor.Refresh()

		for _, autoscalingGroup := range *autoscalingGroups {
			go deathNodeWatcher.CheckIfInstancesToKill(autoscalingGroup)
		}

		notebook.KillAttempt()
		<-ticker.C
	}
}

func initFlags() {

	flag.StringVar(&accessKey, "accessKey", "", "help message for flagname")
	flag.StringVar(&secretKey, "secretKey", "", "help message for flagname")
	flag.StringVar(&region, "region", "eu-west-1", "help message for flagname")
	flag.StringVar(&iamRole, "iamRole", "", "help message for flagname")
	flag.StringVar(&iamSession, "iamSession", "", "help message for flagname")

	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
	flag.StringVar(&mesosUrl, "mesosUrl", "", "The URL for Mesos master")

	flag.Var(&autoscalingGroupNames, "autoscalingGroupName", "The autoscaling group name to monitor")
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

	if len(autoscalingGroupNames) < 1 {
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
