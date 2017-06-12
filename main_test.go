package main

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestOneInstanceRemovalWithoutDestroy(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": &[]string{"default", "one_undesired_host"},
			"DescribeAGByName":       &[]string{"default", "one_undesired_host"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall == nil {
		t.Fatal("One instance should have been removed from ASG. Found nil")
	}
	if len(detachInstanceCall) != 1 {
		t.Fatal("One instance should have been removed from ASG. Found incorrect number")
	}

	destroyInstanceCall := awsConn.Requests["TerminateInstance"]
	if destroyInstanceCall != nil {
		t.Fatal("No instance destroy should have been called")
	}
}

func TestTwoInstanceRemovalWithoutDestroy(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": &[]string{"default", "two_undesired_hosts"},
			"DescribeAGByName":       &[]string{"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall == nil {
		t.Fatal("Two instances should have been removed from ASG. Found nil")
	}
	if len(detachInstanceCall) != 2 {
		t.Fatal("Two instances should have been removed from ASG. Found incorrect number")
	}
	if detachInstanceCall[0][1] == detachInstanceCall[1][1] {
		t.Fatal("Two instance deatch has been called, but all for the same host")
	}
	destroyInstanceCall := awsConn.Requests["TerminateInstance"]
	if destroyInstanceCall != nil {
		t.Fatal("No instance destroy should have been called")
	}
}

func TestTwoInstanceRemovalWithDestroy(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": &[]string{"default", "two_undesired_hosts", "default"},
			"DescribeAGByName":       &[]string{"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "notasks"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn, 0)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	destroyInstanceCall := awsConn.Requests["TerminateInstance"]
	if destroyInstanceCall == nil {
		t.Fatal("Two instance destroy should have been called. Found nil")
	}
	if len(destroyInstanceCall) != 2 {
		t.Fatal("Two instance destroy should have been called. Found incorrect number")
	}
	if destroyInstanceCall[0][0] == destroyInstanceCall[1][0] {
		t.Fatal("Two instance destroy have been called with the same id")
	}

	err := notebook.DestroyInstancesAttempt()
	destroyInstanceCall = awsConn.Requests["TerminateInstance"]
	if err != nil {
		t.Fatal(err)
	}
	if len(destroyInstanceCall) != 2 {
		t.Fatalf("Incorrect number of destroy calls. Actual: %s, Expected: 2", len(destroyInstanceCall))
	}
}

func TestInstanceDeleteIfDelayDeleteIsSet(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": &[]string{"default", "two_undesired_hosts", "two_undesired_hosts"},
			"DescribeAGByName":       &[]string{"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "notasks"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn, 1)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	destroyInstanceCall := awsConn.Requests["TerminateInstance"]
	if destroyInstanceCall == nil {
		t.Fatal("Two instance destroy should have been called. Found nil")
	}
	if len(destroyInstanceCall) != 1 {
		t.Fatalf("Incorrect number of destroy calls. Actual: %s, Expected: 1", len(destroyInstanceCall))
	}

	time.Sleep(time.Second * 2)
	err := notebook.DestroyInstancesAttempt()
	if err != nil {
		t.Fatal(err)
	}

	destroyInstanceCall = awsConn.Requests["TerminateInstance"]
	if len(destroyInstanceCall) != 2 {
		t.Fatalf("Incorrect number of destroy calls. Actual: %s, Expected: 2", len(destroyInstanceCall))
	}
}

func TestNoInstancesBeingRemovedFromASG(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": &[]string{"default", "default"},
			"DescribeAGByName":       &[]string{"default", "default"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall != nil {
		t.Fatal("One instance should have been removed from ASG. Found nil")
	}
}

func prepareRunParameters(awsConn aws.AwsConnectionInterface, mesosConn mesos.MesosConnectionInterface, delayDeleteSeconds int) (
	*mesos.MesosMonitor, *aws.AutoscalingGroups, *deathnode.DeathNodeWatcher, *deathnode.Notebook) {

	protectedFrameworks := []string{"frameworkname1"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}

	constraintsType := "noContraint"
	recommenderType := "firstAvailableAgent"

	mesosMonitor := mesos.NewMesosMonitor(mesosConn, protectedFrameworks)
	autoscalingGroups, _ := aws.NewAutoscalingGroups(awsConn, autoscalingGroupsNames)
	notebook := deathnode.NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds)
	deathNodeWatcher := deathnode.NewDeathNodeWatcher(notebook, mesosMonitor, constraintsType, recommenderType)
	return mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook
}
