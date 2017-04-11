package deathnode

import (
	"fmt"
	"github.com/alanbover/deathnode/aws"
)

func newRecommender(recommenderType string) (recommender, error) {
	switch recommenderType {
	case "firstAvailableAgent":
		return &firstAvailableAgent{}, nil
	default:
		return nil, fmt.Errorf("Recommender type %v not found", recommenderType)
	}
}

type recommender interface {
	find(mesosAgents []aws.InstanceMonitor) *aws.InstanceMonitor
}

type firstAvailableAgent struct{}

func (c *firstAvailableAgent) find(mesosAgents []aws.InstanceMonitor) *aws.InstanceMonitor {
	return &mesosAgents[0]
}
