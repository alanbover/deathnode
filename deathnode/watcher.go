package deathnode

// Given an autoscaling group, decides which is/are the best agent/s to kill

import (
	"github.com/alanbover/deathnode/monitor"
	log "github.com/sirupsen/logrus"
)

// Watcher stores the enough information for decide, if instances need to be removed, which ones are the best
type Watcher struct {
	notebook          *Notebook
	mesosMonitor      *monitor.MesosMonitor
	constraints       constraint
	recommender       recommender
	autoscalingGroups *monitor.AutoscalingGroupsMonitor
}

// NewWatcher returns a new Watcher object
func NewWatcher(notebook *Notebook, mesosMonitor *monitor.MesosMonitor, autoscalingGroups *monitor.AutoscalingGroupsMonitor, constraintType, recommenderType string) *Watcher {

	contrainsts, err := newConstraint(constraintType)
	if err != nil {
		log.Fatal(err)
	}

	recommender, err := newRecommender(recommenderType)
	if err != nil {
		log.Fatal(err)
	}

	return &Watcher{
		notebook:          notebook,
		mesosMonitor:      mesosMonitor,
		constraints:       contrainsts,
		recommender:       recommender,
		autoscalingGroups: autoscalingGroups,
	}
}

// TagInstancesToBeRemoved finds, if any instances to be removed for an autoscaling group, the best instances to
// kill and marks them to be removed
func (y *Watcher) TagInstancesToBeRemoved(autoscalingMonitor *monitor.AutoscalingGroupMonitor) error {

	numUndesiredInstances := autoscalingMonitor.NumUndesiredInstances()
	log.Debugf("Undesired Mesos Agents: %d", numUndesiredInstances)

	removedInstances := 0

	for removedInstances < numUndesiredInstances {
		allowedInstancesToKill := y.constraints.filter(autoscalingMonitor.GetInstances(), y.mesosMonitor)
		bestInstanceToKill := y.recommender.find(allowedInstancesToKill)
		log.Debugf("Mark instance %s for removal", *bestInstanceToKill.GetInstanceID())
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

// Run starts the process of check instances to be killed and try to kill them for all Autoscalings
func (y *Watcher) Run() {

	log.Debug("New check triggered")
	// Refresh autoscaling monitors and mesos monitor
	y.autoscalingGroups.Refresh()
	y.mesosMonitor.Refresh()

	// For each autoscaling monitor, check if any instances needs to be removed
	for _, autoscalingGroup := range y.autoscalingGroups.GetAllMonitors() {
		y.TagInstancesToBeRemoved(autoscalingGroup)
	}

	// Check if any agents are drained, so we can remove them from AWS
	y.DestroyInstancesAttempt()
}
