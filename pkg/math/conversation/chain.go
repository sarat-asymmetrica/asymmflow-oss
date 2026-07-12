// Package conversation tracks prompt conversations as quaternion trajectories.
//
// Om Lokah Samastah Sukhino Bhavantu
package conversation

import (
	"math"

	"ph_holdings_app/pkg/math/encoding"
	"ph_holdings_app/pkg/math/quaternion"
	"ph_holdings_app/pkg/math/trident"
	"ph_holdings_app/pkg/math/vedic"
)

// ConversationChain tracks conversation state as a quaternion trajectory on S3.
type ConversationChain struct {
	waypoints []quaternion.Quaternion
	state     quaternion.Quaternion
	drChain   []int64
	regimes   []trident.Regime
	messages  []string
	momentum  float64
}

// NewConversationChain creates a chain starting at the identity quaternion.
func NewConversationChain() *ConversationChain {
	return &ConversationChain{
		waypoints: make([]quaternion.Quaternion, 0, 32),
		state:     quaternion.Identity(),
		drChain:   make([]int64, 0, 32),
		regimes:   make([]trident.Regime, 0, 32),
		messages:  make([]string, 0, 32),
	}
}

// AddMessage encodes a prompt as a quaternion and SLERPs the state toward it.
func (cc *ConversationChain) AddMessage(prompt string) {
	q := trident.PromptToQuaternion(prompt)

	weight := 1.0 / float64(1+len(cc.waypoints))
	cc.state = quaternion.Slerp(cc.state, q, weight)

	cc.waypoints = append(cc.waypoints, q)
	cc.drChain = append(cc.drChain, trident.ComputeDRSignature(prompt))
	cc.regimes = append(cc.regimes, trident.ClassifyRegime(prompt))
	cc.messages = append(cc.messages, prompt)

	if len(cc.waypoints) >= 2 {
		prev := cc.waypoints[len(cc.waypoints)-2]
		cc.momentum = q.GeodesicDistance(prev) / math.Pi
	}
}

// State returns the current conversation quaternion on S3.
func (cc *ConversationChain) State() quaternion.Quaternion {
	return cc.state
}

// Length returns the number of messages in the chain.
func (cc *ConversationChain) Length() int {
	return len(cc.waypoints)
}

// TotalDistance returns the total geodesic distance traveled along the trajectory.
func (cc *ConversationChain) TotalDistance() float64 {
	if len(cc.waypoints) < 2 {
		return 0
	}
	total := 0.0
	for i := 1; i < len(cc.waypoints); i++ {
		total += cc.waypoints[i-1].GeodesicDistance(cc.waypoints[i])
	}
	return total
}

// CoherenceScore returns how focused the conversation is on [0, 1].
func (cc *ConversationChain) CoherenceScore() float64 {
	if len(cc.waypoints) < 2 {
		return 1.0
	}
	avgDist := cc.TotalDistance() / float64(len(cc.waypoints)-1)
	coherence := 1.0 - (avgDist / math.Pi)
	if coherence < 0 {
		return 0
	}
	return coherence
}

// Momentum returns how fast the conversation is drifting.
func (cc *ConversationChain) Momentum() float64 {
	return cc.momentum
}

// CompositeDR returns the batch-composed DR of all messages.
func (cc *ConversationChain) CompositeDR() int64 {
	return vedic.DigitalRootChain(cc.drChain)
}

// RegimeDrift detects whether the latest message shifted to a different regime.
func (cc *ConversationChain) RegimeDrift() (current trident.Regime, shifted bool, previous trident.Regime) {
	if len(cc.regimes) < 2 {
		if len(cc.regimes) == 1 {
			return cc.regimes[0], false, cc.regimes[0]
		}
		return trident.RegimeStabilization, false, trident.RegimeStabilization
	}
	current = cc.regimes[len(cc.regimes)-1]
	previous = cc.regimes[len(cc.regimes)-2]
	return current, current != previous, previous
}

// DominantRegime returns the most frequent regime across the conversation.
func (cc *ConversationChain) DominantRegime() trident.Regime {
	counts := [3]int{}
	for _, r := range cc.regimes {
		if int(r) >= 0 && int(r) < 3 {
			counts[r]++
		}
	}
	maxIdx := 0
	for i := 1; i < 3; i++ {
		if counts[i] > counts[maxIdx] {
			maxIdx = i
		}
	}
	return trident.Regime(maxIdx)
}

// SuggestTemperature returns an optimal temperature from conversation trajectory.
func (cc *ConversationChain) SuggestTemperature() float64 {
	if len(cc.waypoints) == 0 {
		return 0.5
	}

	base := 0.5
	switch cc.DominantRegime() {
	case trident.RegimeExploration:
		base = 0.8
	case trident.RegimeOptimization:
		base = 0.1
	case trident.RegimeStabilization:
		base = 0.3
	}

	return base + cc.momentum*0.2
}

// StateVerified returns true if the current state is a valid unit quaternion.
func (cc *ConversationChain) StateVerified() bool {
	return cc.state.IsUnit(0.001)
}

// CodonDistanceToLast returns the lossless codon distance between the last two messages.
func (cc *ConversationChain) CodonDistanceToLast() float64 {
	if len(cc.messages) < 2 {
		return 0.0
	}
	return encoding.PromptCodonDistance(
		cc.messages[len(cc.messages)-2],
		cc.messages[len(cc.messages)-1],
	)
}

// DistanceAgreement returns whether semantic and syntactic distances agree.
func (cc *ConversationChain) DistanceAgreement() (agree bool, semantic, syntactic float64) {
	if len(cc.messages) < 2 {
		return true, 0, 0
	}

	semantic = cc.waypoints[len(cc.waypoints)-1].GeodesicDistance(cc.waypoints[len(cc.waypoints)-2])
	syntactic = encoding.PromptCodonDistance(
		cc.messages[len(cc.messages)-2],
		cc.messages[len(cc.messages)-1],
	)

	semNorm := semantic / math.Pi
	synNorm := syntactic / 6.0
	agree = (semNorm > 0.3 && synNorm > 0.3) || (semNorm <= 0.3 && synNorm <= 0.3)
	return
}
