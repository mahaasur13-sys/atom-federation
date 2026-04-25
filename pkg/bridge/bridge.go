// Package bridge — GoA + REDEREF + DESC integration layer.
//
// Integration flow (closed loop):
//
//	GoA.Dispatch(task)
//	  → REDEREF.SelectAgent(belief)   // Thompson sampling
//	    → BudgetEnforcer.Check()       // gate before dispatch
//	      → AgentGraph.Route(message)  // dispatch to agent
//	        → EventStore.Record(event) // DESC event log
//	          → BeliefState.Update()   // Bayesian belief update
//	            → next GoA.Dispatch uses updated belief
package bridge

import (
	"fmt"
	"sync"

	"../goa"
	"../routerapi"
)

// Bridge — unified GoA + REDEREF + DESC pipeline.
type Bridge struct {
	GoA         *goa.AgentGraph
	REDEREF     *router.ReRouter
	EventStore  *EventStore
	BeliefState *BeliefState
	Budget      *router.BudgetEnforcer
	mu          sync.RWMutex
}

// Config — bridge configuration.
type Config struct {
	Seed                uint64
	Agents              []goa.AgentConfig
	EventStorePath      string
	GoAMu               float64
	GoASigma            float64
	ReflectionThreshold float64
	MaxTokens           int64
	MaxDispatch         int
	ReRouteBudget       int
}

// NewBridge — create fully wired integration bridge.
func NewBridge(cfg Config) *Bridge {
	detRNG := router.NewDeterministicRNG(cfg.Seed)

	es := NewEventStore(cfg.EventStorePath)
	bs := NewBeliefState(detRNG, cfg.Agents)
	be := router.NewBudgetEnforcer(cfg.MaxTokens, cfg.MaxDispatch)

	graph := goa.NewAgentGraph(detRNG, goa.Config{
		Mu:            cfg.GoAMu,
		Sigma:         cfg.GoASigma,
		AggregationFn: goa.WeightedAggregator,
	})

	rr := router.NewReRouter(router.ReRouterConfig{
		RNG:               detRNG,
		BeliefState:       bs,
		ReflectionTrigger: cfg.ReflectionThreshold,
		ReRouteBudget:     cfg.ReRouteBudget,
	})

	return &Bridge{
		GoA:         graph,
		REDEREF:     rr,
		EventStore:  es,
		BeliefState: bs,
		Budget:      be,
	}
}

// DispatchResult — result of a dispatch operation.
type DispatchResult struct {
	Rejected       bool
	Reason         string
	AgentSelected  string
	ReflectionUsed bool
}

// DispatchTask — full closed-loop dispatch through the bridge.
func (b *Bridge) DispatchTask(domain, taskID, prompt string) (string, *DispatchResult) {
	b.mu.Lock()
	defer b.mu.Unlock()

	result := &DispatchResult{}

	// Gate 1: Budget check
	if !b.Budget.Allow(taskID) {
		result.Rejected = true
		result.Reason = "budget_exhausted"
		return "", result
	}

	// Gate 2: Thompson sampling via REDEREF
	belief := b.BeliefState.GetBelief()
	agentID := b.REDEREF.SelectAgent(domain, belief)
	result.AgentSelected = agentID

	// Gate 3: Reflection check
	if b.REDEREF.ShouldReflect() {
		result.ReflectionUsed = true
		agentID = b.REDEREF.ReRoute(domain, belief)
		result.AgentSelected = agentID
	}

	// Route through GoA
	msg := goa.Message{
		Type:      goa.MsgTask,
		TaskID:    taskID,
		Domain:    domain,
		Content:   prompt,
		Sender:    "bridge",
		Recipient: agentID,
	}
	b.GoA.Route(msg)

	// Record to DESC EventStore
	b.EventStore.Record(Event{
		Tick:    b.GoA.GetCurrentTick(),
		Type:    "dispatch",
		AgentID: agentID,
		Domain:  domain,
		TaskID:  taskID,
		Mu:      belief[agentID],
	})

	return agentID, result
}

// RecordOutcome — update beliefs after task completion.
func (b *Bridge) RecordOutcome(taskID, agentID string, success bool, reward float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.EventStore.Record(Event{
		Tick:    b.GoA.GetCurrentTick(),
		Type:    "outcome",
		AgentID: agentID,
		TaskID:  taskID,
		Success: success,
		Reward:  reward,
	})

	b.BeliefState.Update(agentID, success, reward)
	b.REDEREF.UpdateBelief(agentID, success, reward)
}

// GetEventLog — return full DESC event log for replay.
func (b *Bridge) GetEventLog() []Event {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.EventStore.All()
}
