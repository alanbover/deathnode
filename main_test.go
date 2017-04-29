package main

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
	"testing"
)

func TestOneInstanceRemovalWithoutDestroy(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.AwsConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": &[]string{
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeAGByName": &[]string{"default", "one_undesired_host"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)

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
			"DescribeAGByName": &[]string{"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)

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
			"DescribeAGByName": &[]string{"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "notasks"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)

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

	err := notebook.KillAttempt()
	destroyInstanceCall = awsConn.Requests["TerminateInstance"]
	if err != nil {
		t.Fatal(err)
	}
	if len(destroyInstanceCall) != 2 {
		t.Fatal("Instance from notebook was not correctly removed and a new destroy was called")
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
			"DescribeAGByName": &[]string{"default", "default"},
		},
	}

	mesosConn := &mesos.MesosConnectionMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": &[]string{"default", "default"},
			"GetMesosSlaves":     &[]string{"default", "default"},
			"GetMesosTasks":      &[]string{"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn)

	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)
	Run(mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall != nil {
		t.Fatal("One instance should have been removed from ASG. Found nil")
	}
}

func prepareRunParameters(awsConn aws.AwsConnectionInterface, mesosConn mesos.MesosConnectionInterface) (
	*mesos.MesosMonitor, *aws.AutoscalingGroups, *deathnode.DeathNodeWatcher, *deathnode.Notebook) {

	protectedFrameworks := []string{"frameworkname1"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}

	constraintsType := "noContraint"
	recommenderType := "firstAvailableAgent"

	mesosMonitor := mesos.NewMesosMonitor(mesosConn, protectedFrameworks)
	autoscalingGroups, _ := aws.NewAutoscalingGroups(awsConn, autoscalingGroupsNames)
	notebook := deathnode.NewNotebook(mesosMonitor)
	deathNodeWatcher := deathnode.NewDeathNodeWatcher(notebook, mesosMonitor, constraintsType, recommenderType)
	return mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook
}
