package main

import "time"
import "flag"

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	log "github.com/sirupsen/logrus"
)

var accessKey, secretKey, region, iamRole, iamSession, mesosURL string
var debug bool
var pollingSeconds int

func main() {

	ctx := &context.ApplicationContext{}

	initFlags(ctx)
	enforceFlags(ctx)

	log.SetLevel(log.InfoLevel)
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	// Create the monitors for autoscaling groups
	awsConn, err := aws.NewClient(accessKey, secretKey, region, iamRole, iamSession)
	if err != nil {
		log.Fatal("Error connecting to AWS: ", err)
	}
	// Create the Mesos monitor
	mesosConn := &mesos.Client{
		MasterURL: mesosURL,
	}
	ctx.AwsConn = awsConn
	ctx.MesosConn = mesosConn

	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(ctx)
	mesosMonitor := monitor.NewMesosMonitor(ctx)


	// Create deathnoteWatcher
	notebook := deathnode.NewNotebook(ctx, autoscalingGroups, mesosMonitor)
	deathNodeWatcher := deathnode.NewWatcher(
		ctx, notebook, mesosMonitor, autoscalingGroups)

	ticker := time.NewTicker(time.Second * time.Duration(pollingSeconds))
	for {
		go deathNodeWatcher.Run()
		<-ticker.C
	}
}

func initFlags(context *context.ApplicationContext) {

	flag.StringVar(&accessKey, "accessKey", "", "AWS_ACCESS_KEY_ID.")
	flag.StringVar(&secretKey, "secretKey", "", "AWS_SECRET_ACCESS_KEY.")
	flag.StringVar(&region, "region", "eu-west-1", "AWS_REGION.")
	flag.StringVar(&iamRole, "iamRole", "", "IAMROLE to assume.")
	flag.StringVar(&iamSession, "iamSession", "", "Session for IAMROLE.")

	flag.BoolVar(&debug, "debug", false, "Enable debug logging.")
	flag.StringVar(&mesosURL, "mesosUrl", "", "The URL for Mesos master.")

	flag.Var(&context.Conf.AutoscalingGroupPrefixes, "autoscalingGroupName", "An autoscalingGroup prefix for monitor.")
	flag.Var(&context.Conf.ProtectedFrameworks, "protectedFrameworks", "The mesos frameworks to wait for kill the node.")
	flag.Var(&context.Conf.ProtectedTasksLabels, "protectedTaskLabels", "The labels used for protected tasks.")

	flag.StringVar(
		&context.Conf.ConstraintsType, "constraintsType", "noContraint", "The constrainst implementation to use.")
	flag.StringVar(
		&context.Conf.RecommenderType, "recommenderType", "firstAvailableAgent", "The recommender implementation to use.")
	flag.StringVar(
		&context.Conf.DeathNodeMark, "deathNodeMark", "DEATH_NODE_MARK", "The tag to apply for instances to be deleted.")

	flag.IntVar(&pollingSeconds, "polling", 60, "Seconds between executions.")
	flag.IntVar(&context.Conf.DelayDeleteSeconds, "delayDelete", 0, "Time to wait between kill executions (in seconds).")

	flag.Parse()
}

func enforceFlags(context *context.ApplicationContext) {

	if mesosURL == "" {
		flag.Usage()
		log.Fatal("mesosUrl flag is required")
	}

	if len(context.Conf.AutoscalingGroupPrefixes) < 1 {
		flag.Usage()
		log.Fatal("at least one autoscalingGroupName flag is required")
	}

	if len(context.Conf.ProtectedFrameworks) < 1 {
		flag.Usage()
		log.Fatal("at least one registeredFramework flag is required")
	}
}
