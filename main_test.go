package main

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/deathnode"
	"github.com/alanbover/deathnode/mesos"
	"github.com/alanbover/deathnode/monitor"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

func TestOneInstanceRemovalWithoutDestroy(t *testing.T) {

	log.SetLevel(log.DebugLevel)

	awsConn := &aws.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "one_undesired_host"},
			"DescribeAGByName":       {"default", "one_undesired_host"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default"},
			"GetMesosSlaves":     {"default", "default"},
			"GetMesosTasks":      {"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

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

	awsConn := &aws.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "two_undesired_hosts"},
			"DescribeAGByName":       {"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default"},
			"GetMesosSlaves":     {"default", "default"},
			"GetMesosTasks":      {"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall == nil {
		t.Fatal("Two instances should have been removed from ASG. Found nil")
	}
	if len(detachInstanceCall) != 2 {
		t.Fatalf("Incorrect number of detachInstance calls. Actual: %s, Expected: 2", len(detachInstanceCall))
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

	awsConn := &aws.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "two_undesired_hosts", "default"},
			"DescribeAGByName":       {"default", "two_undesired_hosts"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default"},
			"GetMesosSlaves":     {"default", "default"},
			"GetMesosTasks":      {"default", "notasks"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn, 0)

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

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

	awsConn := &aws.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"node1", "node2", "node3",
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "two_undesired_hosts", "two_undesired_hosts_one_removed", "two_undesired_hosts_one_removed"},
			"DescribeAGByName":       {"default", "two_undesired_hosts", "two_undesired_hosts_one_removed"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default", "default"},
			"GetMesosSlaves":     {"default", "default", "default"},
			"GetMesosTasks":      {"default", "notasks", "notasks"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook := prepareRunParameters(awsConn, mesosConn, 1)

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if len(detachInstanceCall) != 2 {
		t.Fatalf("Incorrect number of detachInstance calls. Actual: %s, Expected: 2", len(detachInstanceCall))
	}

	setTagInstanceCall := awsConn.Requests["SetInstanceTag"]
	if len(setTagInstanceCall) != 2 {
		t.Fatalf("Incorrect number of setTagInstanceCall calls. Actual: %s, Expected: 2", len(setTagInstanceCall))
	}
	if setTagInstanceCall[0][2] == setTagInstanceCall[1][2] {
		t.Fatal("setTagInstance called two times for the same instance id", len(setTagInstanceCall))
	}

	destroyInstanceCall := awsConn.Requests["TerminateInstance"]
	if destroyInstanceCall == nil {
		t.Fatal("Two instance destroy should have been called. Found nil")
	}
	if len(destroyInstanceCall) != 1 {
		t.Fatalf("Incorrect number of destroy calls. Actual: %s, Expected: 1", len(destroyInstanceCall))
	}

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	detachInstanceCall = awsConn.Requests["DetachInstance"]
	if len(detachInstanceCall) != 2 {
		t.Fatalf("Incorrect number of detachInstance calls. Actual: %s, Expected: 2", len(detachInstanceCall))
	}

	setTagInstanceCall = awsConn.Requests["SetInstanceTag"]
	if len(setTagInstanceCall) != 2 {
		t.Fatalf("Incorrect number of setTagInstanceCall calls. Actual: %s, Expected: 2", len(setTagInstanceCall))
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

	awsConn := &aws.ConnectionMock{
		Records: map[string]*[]string{
			"DescribeInstanceById": {
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "default"},
			"DescribeAGByName":       {"default", "default"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default"},
			"GetMesosSlaves":     {"default", "default"},
			"GetMesosTasks":      {"default", "default"},
		},
	}

	mesosMonitor, autoscalingGroups, deathNodeWatcher, _ := prepareRunParameters(awsConn, mesosConn, 0)

	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)
	run(mesosMonitor, autoscalingGroups, deathNodeWatcher)

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall != nil {
		t.Fatal("One instance should have been removed from ASG. Found nil")
	}
}

func prepareRunParameters(awsConn aws.ClientInterface, mesosConn mesos.ClientInterface, delayDeleteSeconds int) (
	*monitor.MesosMonitor, *monitor.AutoscalingGroupsMonitor, *deathnode.Watcher, *deathnode.Notebook) {

	protectedFrameworks := []string{"frameworkName1"}
	protectedTasksLabels := []string{"task1"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}

	constraintsType := "noContraint"
	recommenderType := "smallestInstanceId"

	mesosMonitor := monitor.NewMesosMonitor(mesosConn, protectedFrameworks, protectedTasksLabels)
	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, autoscalingGroupsNames, "DEATH_NODE_MARK")
	notebook := deathnode.NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds, "DEATH_NODE_MARK")
	deathNodeWatcher := deathnode.NewWatcher(notebook, mesosMonitor, constraintsType, recommenderType)
	return mesosMonitor, autoscalingGroups, deathNodeWatcher, notebook
}
