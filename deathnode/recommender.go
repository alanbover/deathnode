package deathnode

import (
	"fmt"
	"github.com/alanbover/deathnode/monitor"
	"strings"
)

func newRecommender(recommenderType string) (recommender, error) {
	switch recommenderType {
	case "firstAvailableAgent":
		return &firstAvailableAgent{}, nil
	case "smallestInstanceId":
		return &smallestInstanceID{}, nil
	default:
		return nil, fmt.Errorf("Recommender type %v not found", recommenderType)
	}
}

type recommender interface {
	find(mesosAgents []*monitor.InstanceMonitor) *monitor.InstanceMonitor
}

type firstAvailableAgent struct{}

func (c *firstAvailableAgent) find(mesosAgents []*monitor.InstanceMonitor) *monitor.InstanceMonitor {
	return mesosAgents[0]
}

type smallestInstanceID struct{}

func (c *smallestInstanceID) find(mesosAgents []*monitor.InstanceMonitor) *monitor.InstanceMonitor {
	mesosAgentSmallestInstanceID := mesosAgents[0]
	for _, mesosAgent := range mesosAgents {
		if strings.Compare(*mesosAgent.GetInstanceID(), *mesosAgentSmallestInstanceID.GetInstanceID()) < 0 {
			mesosAgentSmallestInstanceID = mesosAgent
		}
	}

	return mesosAgentSmallestInstanceID
}
