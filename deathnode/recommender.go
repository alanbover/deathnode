package deathnode

import (
	"fmt"
	"github.com/alanbover/deathnode/aws"
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
	find(mesosAgents []*aws.InstanceMonitor) *aws.InstanceMonitor
}

type firstAvailableAgent struct{}

func (c *firstAvailableAgent) find(mesosAgents []*aws.InstanceMonitor) *aws.InstanceMonitor {
	return mesosAgents[0]
}

type smallestInstanceID struct{}

func (c *smallestInstanceID) find(mesosAgents []*aws.InstanceMonitor) *aws.InstanceMonitor {
	mesosAgentSmallestInstanceID := mesosAgents[0]
	for _, mesosAgent := range mesosAgents {
		if strings.Compare(mesosAgent.GetInstanceID(), mesosAgentSmallestInstanceID.GetInstanceID()) < 0 {
			mesosAgentSmallestInstanceID = mesosAgent
		}
	}

	return mesosAgentSmallestInstanceID
}
