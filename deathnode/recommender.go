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
		return &smallestInstanceId{}, nil
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

type smallestInstanceId struct{}

func (c *smallestInstanceId) find(mesosAgents []*aws.InstanceMonitor) *aws.InstanceMonitor {
	mesosAgentSmallestInstanceId := mesosAgents[0]
	for _, mesosAgent := range mesosAgents {
		if strings.Compare(mesosAgent.GetInstanceId(), mesosAgentSmallestInstanceId.GetInstanceId()) < 0 {
			mesosAgentSmallestInstanceId = mesosAgent
		}
	}

	return mesosAgentSmallestInstanceId
}
