package deathnode

// Given an autoscaling group, decides which is/are the best agent/s to kill

import (
	"github.com/alanbover/deathnode/context"
	"github.com/alanbover/deathnode/monitor"
	log "github.com/sirupsen/logrus"
)

// Watcher stores the enough information for decide, if instances need to be removed, which ones are the best
type Watcher struct {
	notebook                  *Notebook
	mesosMonitor              *monitor.MesosMonitor
	autoscalingServiceMonitor *monitor.AutoscalingServiceMonitor
	constraints               []constraint
	recommender               recommender
}

// NewWatcher returns a new Watcher object
func NewWatcher(ctx *context.ApplicationContext) *Watcher {

	autoscalingServiceMonitor := monitor.NewAutoscalingServiceMonitor(ctx)
	mesosMonitor := monitor.NewMesosMonitor(ctx)

	constraints := []constraint{}
	for _, constraint := range ctx.Conf.ConstraintsType {
		newConstraint, err := newConstraint(constraint)
		if err != nil {
			log.Fatal(err)
		}
		constraints = append(constraints, newConstraint)
	}

	recommender, err := newRecommender(ctx.Conf.RecommenderType)
	if err != nil {
		log.Fatal(err)
	}

	return &Watcher{
		notebook:                  NewNotebook(ctx, autoscalingServiceMonitor, mesosMonitor),
		mesosMonitor:              mesosMonitor,
		constraints:               constraints,
		recommender:               recommender,
		autoscalingServiceMonitor: autoscalingServiceMonitor,
	}
}

// TagInstancesToBeRemoved finds, if any instances to be removed for an autoscaling group, the best instances to
// kill and tags them to be removed
func (y *Watcher) TagInstancesToBeRemoved(autoscalingMonitor *monitor.AutoscalingGroupMonitor) {

	numUndesiredInstances := autoscalingMonitor.GetNumUndesiredInstances()
	log.Debugf("Undesired Mesos Agents: %d", numUndesiredInstances)

	for removedInstances := 0; removedInstances < numUndesiredInstances; removedInstances++ {

		allowedInstances := autoscalingMonitor.GetInstances()
		for _, constraint := range y.constraints {
			allowedInstances = constraint.filter(allowedInstances, y.mesosMonitor)
		}
		bestInstance := y.recommender.find(allowedInstances)

		log.Debugf("Tagging instance %s for removal", *bestInstance.InstanceID())
		if err := bestInstance.TagToBeRemoved(); err != nil {
			log.Errorf("Unable to tag instance %s for removal", bestInstance.IP())
			log.Error(err)
			break
		}
	}
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

	y.autoscalingServiceMonitor.Refresh()
	y.mesosMonitor.Refresh()

	for _, autoscalingGroup := range y.autoscalingServiceMonitor.GetAutoscalingGroupMonitorsList() {
		y.TagInstancesToBeRemoved(autoscalingGroup)
	}

	y.DestroyInstancesAttempt()
}
