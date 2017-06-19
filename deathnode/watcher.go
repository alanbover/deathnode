package deathnode

// Given an autoscaling group, decides which is/are the best agent/s to kill

import (
	"github.com/alanbover/deathnode/aws"
	"github.com/alanbover/deathnode/mesos"
	log "github.com/sirupsen/logrus"
)

// Watcher stores the enough information for decide, if instances need to be removed, which ones are the best
type Watcher struct {
	notebook     *Notebook
	mesosMonitor *mesos.Monitor
	constraints  constraint
	recommender  recommender
}

// NewWatcher returns a new Watcher object
func NewWatcher(notebook *Notebook, mesosMonitor *mesos.Monitor, constraintType, recommenderType string) *Watcher {

	contrainsts, err := newConstraint(constraintType)
	if err != nil {
		log.Fatal(err)
	}

	recommender, err := newRecommender(recommenderType)
	if err != nil {
		log.Fatal(err)
	}

	return &Watcher{
		notebook:     notebook,
		mesosMonitor: mesosMonitor,
		constraints:  contrainsts,
		recommender:  recommender,
	}
}

// RemoveUndesiredInstances finds, if any instances to be removed for an autoscaling group, the best instances to
// kill and marks them to be removed
func (y *Watcher) RemoveUndesiredInstances(autoscalingMonitor *aws.AutoscalingGroupMonitor) error {

	numUndesiredInstances := autoscalingMonitor.NumUndesiredInstances()
	log.Debugf("Undesired Mesos Agents: %d", numUndesiredInstances)

	removedInstances := 0

	for removedInstances < numUndesiredInstances {
		allowedInstancesToKill := y.constraints.filter(autoscalingMonitor.GetInstancesNotMarkedToBeRemoved())
		bestInstanceToKill := y.recommender.find(allowedInstancesToKill)
		log.Debugf("Mark instance %s for removal", bestInstanceToKill.GetInstanceID())
		err := bestInstanceToKill.MarkToBeRemoved()
		if err != nil {
			log.Errorf("Unable to mark instance %s for removal", bestInstanceToKill.GetIP())
			log.Error(err)
			break
		}

		removedInstances++
	}

	return nil
}

// DestroyInstancesAttempt try for those instances marked to be deleted to delete them
func (y *Watcher) DestroyInstancesAttempt() {

	err := y.notebook.DestroyInstancesAttempt()
	if err != nil {
		log.Error(err)
	}
}
