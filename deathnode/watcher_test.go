package deathnode

import (
	"fmt"
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/mesos"
	"github.com/benbjohnson/clock"
	"testing"
)

type testCollectionValues struct {
	awsConn            *aws.ConnectionMock
	mesosConn          *mesos.ClientMock
	delayDeleteSeconds int
	times              int
}

type expectedResult struct {
	values                            testCollectionValues
	numInstancesRemoved               int
	numMarkToBeRemoved                int
	numRemovedInstancesProtection     int
	numRecordLifecycleActionHeartbeat int
}

func TestWatcher(t *testing.T) {

	expectedResults := []expectedResult{
		{
			values: testCollectionValues{
				awsConn: &aws.ConnectionMock{
					Records: map[string]*[]string{
						"DescribeInstanceById": {
							"node1", "node2", "node3",
							"node1", "node2", "node3",
						},
						"DescribeInstancesByTag": {"default", "default"},
						"DescribeAGByName":       {"default", "default"},
					},
				},
				mesosConn: &mesos.ClientMock{
					Records: map[string]*[]string{
						"GetMesosFrameworks": {"default", "default"},
						"GetMesosSlaves":     {"default", "default"},
						"GetMesosTasks":      {"default", "default"},
					},
				},
				delayDeleteSeconds: 0,
				times:              2,
			},
			numInstancesRemoved:               0,
			numMarkToBeRemoved:                0,
			numRemovedInstancesProtection:     0,
			numRecordLifecycleActionHeartbeat: 0,
		},
		{
			values: testCollectionValues{
				awsConn: &aws.ConnectionMock{
					Records: map[string]*[]string{
						"DescribeInstanceById": {
							"node1", "node2", "node3",
							"node1", "node2", "node3",
						},
						"DescribeInstancesByTag": {"default", "one_undesired_host"},
						"DescribeAGByName":       {"default", "one_undesired_host"},
					},
				},
				mesosConn: &mesos.ClientMock{
					Records: map[string]*[]string{
						"GetMesosFrameworks": {"default", "default"},
						"GetMesosSlaves":     {"default", "default"},
						"GetMesosTasks":      {"notasks", "notasks"},
					},
				},
				delayDeleteSeconds: 0,
				times:              2,
			},
			numInstancesRemoved:               0,
			numMarkToBeRemoved:                1,
			numRemovedInstancesProtection:     1,
			numRecordLifecycleActionHeartbeat: 0,
		},
		{
			values: testCollectionValues{
				awsConn: &aws.ConnectionMock{
					Records: map[string]*[]string{
						"DescribeInstanceById": {
							"node1", "node2", "node3",
							"node1", "node2", "node3",
						},
						"DescribeInstancesByTag": {"default", "two_undesired_hosts"},
						"DescribeAGByName":       {"default", "two_undesired_hosts"},
					},
				},
				mesosConn: &mesos.ClientMock{
					Records: map[string]*[]string{
						"GetMesosFrameworks": {"default", "default"},
						"GetMesosSlaves":     {"default", "default"},
						"GetMesosTasks":      {"notasks", "notasks"},
					},
				},
				delayDeleteSeconds: 0,
				times:              2,
			},
			numInstancesRemoved:               0,
			numMarkToBeRemoved:                2,
			numRemovedInstancesProtection:     2,
			numRecordLifecycleActionHeartbeat: 0,
		},
		{
			values: testCollectionValues{
				awsConn: &aws.ConnectionMock{
					Records: map[string]*[]string{
						"DescribeInstanceById": {
							"node1", "node2", "node3",
							"node1", "node2", "node3",
						},
						"DescribeInstancesByTag": {"default", "two_undesired_hosts",
							"two_undesired_hosts"},
						"DescribeAGByName": {"default", "two_undesired_hosts",
							"two_undesired_hosts_two_terminating"},
					},
				},
				mesosConn: &mesos.ClientMock{
					Records: map[string]*[]string{
						"GetMesosFrameworks": {"default", "default", "default"},
						"GetMesosSlaves":     {"default", "default", "default"},
						"GetMesosTasks":      {"notasks", "notasks", "notasks"},
					},
				},
				delayDeleteSeconds: 0,
				times:              3,
			},
			numInstancesRemoved:               2,
			numMarkToBeRemoved:                2,
			numRemovedInstancesProtection:     2,
			numRecordLifecycleActionHeartbeat: 0,
		},
		{
			values: testCollectionValues{
				awsConn: &aws.ConnectionMock{
					Records: map[string]*[]string{
						"DescribeInstanceById": {
							"node1", "node2", "node3",
							"node1", "node2", "node3",
						},
						"DescribeInstancesByTag": {"default", "two_undesired_hosts",
							"two_undesired_hosts"},
						"DescribeAGByName": {
							"default", "two_undesired_hosts",
							"two_undesired_hosts_two_terminating"},
					},
				},
				mesosConn: &mesos.ClientMock{
					Records: map[string]*[]string{
						"GetMesosFrameworks": {"default", "default", "default"},
						"GetMesosSlaves":     {"default", "default", "default"},
						"GetMesosTasks":      {"notasks", "notasks", "notasks"},
					},
				},
				delayDeleteSeconds: 1,
				times:              3,
			},
			numInstancesRemoved:               1,
			numMarkToBeRemoved:                2,
			numRemovedInstancesProtection:     2,
			numRecordLifecycleActionHeartbeat: 0,
		},
	}

	for i, result := range expectedResults {
		t.Run(fmt.Sprintf("TestWatcher resultValue %v", i), func(t *testing.T) {
			runWatcher(result.values)
			requests := result.values.awsConn.Requests

			// Check number of instance protection removals
			if result.numRemovedInstancesProtection > 0 && requests["RemoveASGInstanceProtection"] == nil {
				t.Fatalf("Incorrect number of InstanceProtection removal. Expected: %v, Found: nil",
					result.numRemovedInstancesProtection)
			}
			if result.numRemovedInstancesProtection == 0 && requests["RemoveASGInstanceProtection"] != nil {
				t.Fatalf("Incorrect number of InstanceProtection removal. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["RemoveASGInstanceProtection"]))
			}
			if result.numRemovedInstancesProtection != len(requests["RemoveASGInstanceProtection"]) {
				t.Fatalf("Incorrect number of InstanceProtection removal. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["RemoveASGInstanceProtection"]))
			}

			// Check number of instances marked to be removed
			if result.numMarkToBeRemoved > 0 && requests["SetInstanceTag"] == nil {
				t.Fatalf("Incorrect number of instances marked to be removed. Expected: %v, Found: nil",
					result.numRemovedInstancesProtection)
			}
			if result.numMarkToBeRemoved == 0 && requests["SetInstanceTag"] != nil {
				t.Fatalf("Incorrect number of instances marked to be removed. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["SetInstanceTag"]))
			}
			if result.numMarkToBeRemoved != len(requests["SetInstanceTag"]) {
				t.Fatalf("Incorrect number of instances marked to be removed. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["SetInstanceTag"]))
			}

			// Check number of instances removed
			if result.numInstancesRemoved > 0 && requests["CompleteLifecycleAction"] == nil {
				t.Fatalf("Incorrect number of instances removed. Expected: %v, Found: nil",
					result.numRemovedInstancesProtection)
			}
			if result.numInstancesRemoved == 0 && requests["CompleteLifecycleAction"] != nil {
				t.Fatalf("Incorrect number of instances removed. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["CompleteLifecycleAction"]))
			}
			if result.numInstancesRemoved != len(requests["CompleteLifecycleAction"]) {
				t.Fatalf("Incorrect number of instances removed. Expected: %v, Found: %v",
					result.numRemovedInstancesProtection, len(requests["CompleteLifecycleAction"]))
			}

			// Check number of restarted lifecycle heartbeats
			if result.numRecordLifecycleActionHeartbeat > 0 && requests["RecordLifecycleActionHeartbeat"] == nil {
				t.Fatalf("Incorrect number of lifecycle heartbeats. Expected: %v, Found: nil",
					result.numRecordLifecycleActionHeartbeat)
			}
			if result.numRecordLifecycleActionHeartbeat == 0 && requests["RecordLifecycleActionHeartbeat"] != nil {
				t.Fatalf("Incorrect number of lifecycle heartbeats. Expected: %v, Found: %v",
					result.numRecordLifecycleActionHeartbeat, len(requests["RecordLifecycleActionHeartbeat"]))
			}
			if result.numRecordLifecycleActionHeartbeat != len(requests["RecordLifecycleActionHeartbeat"]) {
				t.Fatalf("Incorrect number of lifecycle heartbeats. Expected: %v, Found: %v",
					result.numRecordLifecycleActionHeartbeat, len(requests["RecordLifecycleActionHeartbeat"]))
			}
		})
	}
}

func newWatcher(testValues testCollectionValues) *Watcher {

	ctx := &context.ApplicationContext{
		Clock:     clock.New(),
		AwsConn:   testValues.awsConn,
		MesosConn: testValues.mesosConn,
		Conf: context.ApplicationConf{
			DeathNodeMark:            "DEATH_NODE_MARK",
			AutoscalingGroupPrefixes: []string{"some-Autoscaling-Group"},
			ProtectedFrameworks:      []string{"frameworkName1"},
			ProtectedTasksLabels:     []string{"DEATHNODE_PROTECTED"},
			DelayDeleteSeconds:       testValues.delayDeleteSeconds,
			ConstraintsType:          "noContraint",
			RecommenderType:          "smallestInstanceId",
		},
	}

	deathNodeWatcher := NewWatcher(ctx)
	return deathNodeWatcher
}

func runWatcher(testValues testCollectionValues) {

	watcher := newWatcher(testValues)
	for iter := 0; iter < testValues.times; iter++ {
		watcher.Run()
	}
}
