package deathnode

import (
	"github.com/alanbover/deathnode/aws"
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

	deathNodeWatcher := newWatcher(awsConn, mesosConn, 0)

	deathNodeWatcher.Run()
	deathNodeWatcher.Run()

	removeInstanceProtectionCall := awsConn.Requests["RemoveASGInstanceProtection"]
	if removeInstanceProtectionCall == nil {
		t.Fatal("Should remove instance protection. Found nil")
	}
	if len(removeInstanceProtectionCall) != 1 {
		t.Fatal("One instance should have been removed from ASG. Found incorrect number")
	}

	completeLifecycleHookCall := awsConn.Requests["CompleteLifecycleAction"]
	if completeLifecycleHookCall != nil {
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

	deathNodeWatcher := newWatcher(awsConn, mesosConn, 0)

	deathNodeWatcher.Run()
	deathNodeWatcher.Run()

	removeInstanceProtectionCall := awsConn.Requests["RemoveASGInstanceProtection"]
	if removeInstanceProtectionCall == nil {
		t.Fatal("Two instances should have been removed from ASG. Found nil")
	}
	if len(removeInstanceProtectionCall) != 2 {
		t.Fatalf("Incorrect number of detachInstance calls. Actual: %s, Expected: 2", len(removeInstanceProtectionCall))
	}
	if removeInstanceProtectionCall[0][1] == removeInstanceProtectionCall[1][1] {
		t.Fatal("Two instance deatch has been called, but all for the same host")
	}

	destroyInstanceCall := awsConn.Requests["CompleteLifecycleAction"]
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
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "two_undesired_hosts", "two_undesired_hosts", "two_undesired_hosts"},
			"DescribeAGByName":       {"default", "two_undesired_hosts", "two_undesired_hosts_two_terminating"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default", "default"},
			"GetMesosSlaves":     {"default", "default", "default"},
			"GetMesosTasks":      {"default", "notasks", "notasks"},
		},
	}

	deathNodeWatcher := newWatcher(awsConn, mesosConn, 0)

	deathNodeWatcher.Run()
	deathNodeWatcher.Run()
	deathNodeWatcher.Run()

	destroyInstanceCall := awsConn.Requests["CompleteLifecycleAction"]
	if destroyInstanceCall == nil {
		t.Fatal("Two instance destroy should have been called. Found nil")
	}
	if len(destroyInstanceCall) != 2 {
		t.Fatal("Two instance destroy should have been called. Found incorrect number")
	}

	if destroyInstanceCall[0][1] == destroyInstanceCall[1][1] {
		t.Fatal("Two instance destroy have been called with the same id")
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
				"node1", "node2", "node3",
				"node1", "node2", "node3",
			},
			"DescribeInstancesByTag": {"default", "two_undesired_hosts",
				"two_undesired_hosts", "two_undesired_hosts", "two_undesired_hosts"},
			"DescribeAGByName": {"default", "two_undesired_hosts", "two_undesired_hosts_two_terminating",
				"two_undesired_hosts_two_terminating", "two_undesired_hosts_two_terminating"},
		},
	}

	mesosConn := &mesos.ClientMock{
		Records: map[string]*[]string{
			"GetMesosFrameworks": {"default", "default", "default", "default", "default"},
			"GetMesosSlaves":     {"default", "default", "default", "default", "default"},
			"GetMesosTasks":      {"default", "notasks", "notasks", "notasks", "notasks"},
		},
	}

	deathNodeWatcher := newWatcher(awsConn, mesosConn, 1)

	deathNodeWatcher.Run()
	deathNodeWatcher.Run()
	deathNodeWatcher.Run()

	detachInstanceCall := awsConn.Requests["RemoveASGInstanceProtection"]
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

	destroyInstanceCall := awsConn.Requests["CompleteLifecycleAction"]
	if destroyInstanceCall == nil {
		t.Fatal("Two instance destroy should have been called. Found nil")
	}
	if len(destroyInstanceCall) != 1 {
		t.Fatalf("Incorrect number of destroy calls. Actual: %s, Expected: 1", len(destroyInstanceCall))
	}

	deathNodeWatcher.Run()
	detachInstanceCall = awsConn.Requests["RemoveASGInstanceProtection"]
	if len(detachInstanceCall) != 2 {
		t.Fatalf("Incorrect number of detachInstance calls. Actual: %s, Expected: 2", len(detachInstanceCall))
	}

	setTagInstanceCall = awsConn.Requests["SetInstanceTag"]
	if len(setTagInstanceCall) != 2 {
		t.Fatalf("Incorrect number of setTagInstanceCall calls. Actual: %s, Expected: 2", len(setTagInstanceCall))
	}

	time.Sleep(time.Second * 2)
	deathNodeWatcher.Run()

	destroyInstanceCall = awsConn.Requests["CompleteLifecycleAction"]
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

	deathNodeWatcher := newWatcher(awsConn, mesosConn, 0)

	deathNodeWatcher.Run()
	deathNodeWatcher.Run()

	detachInstanceCall := awsConn.Requests["DetachInstance"]
	if detachInstanceCall != nil {
		t.Fatal("One instance should have been removed from ASG. Found nil")
	}
}

func newWatcher(awsConn aws.ClientInterface, mesosConn mesos.ClientInterface, delayDeleteSeconds int) *Watcher {

	protectedFrameworks := []string{"frameworkName1"}
	protectedTasksLabels := []string{"DEATHNODE_PROTECTED"}
	autoscalingGroupsNames := []string{"some-Autoscaling-Group"}

	constraintsType := "noContraint"
	recommenderType := "smallestInstanceId"

	mesosMonitor := monitor.NewMesosMonitor(mesosConn, protectedFrameworks, protectedTasksLabels)
	autoscalingGroups, _ := monitor.NewAutoscalingGroupMonitors(awsConn, autoscalingGroupsNames, "DEATH_NODE_MARK")
	notebook := NewNotebook(autoscalingGroups, awsConn, mesosMonitor, delayDeleteSeconds, "DEATH_NODE_MARK")
	deathNodeWatcher := NewWatcher(notebook, mesosMonitor, autoscalingGroups, constraintsType, recommenderType)
	return deathNodeWatcher
}
