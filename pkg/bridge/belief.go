package bridge

import "math"

// BeliefState — shared belief for REDEREF Thompson sampling.
type BeliefState struct {
	rng   *DeterministicRNG
	mu    map[string]float64
	sigma map[string]float64
}

// DeterministicRNG — deterministic RNG for reproducible runs.
type DeterministicRNG struct {
	seed    uint64
	counter uint64
}

func NewDeterministicRNG(seed uint64) *DeterministicRNG {
	return &DeterministicRNG{seed: seed}
}

func (r *DeterministicRNG) Float64Range(min, max float64) float64 {
	h := fnvHash(r.seed, r.counter)
	r.counter++
	norm := float64(h) / float64(^uint64(0))
	return min + norm*(max-min)
}

func (r *DeterministicRNG) GetSeed() uint64 { return r.seed }
func (r *DeterministicRNG) Reset()         { r.counter = 0 }

func (r *DeterministicRNG) NormalSample(mu, sigma float64) float64 {
	u1 := r.Float64Range(0.0001, 1.0)
	u2 := r.Float64Range(0.0001, 1.0)
	z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return mu + sigma*z
}

func fnvHash(seed, counter uint64) uint64 {
	h := uint64(14695981039346656037)
	h ^= seed
	h *= 1099511628211
	h ^= counter
	h *= 1099511628211
	return h
}

// NewBeliefState — initialize with uniform priors.
func NewBeliefState(rng *DeterministicRNG, agents []goa.AgentConfig) *BeliefState {
	bs := &BeliefState{rng: rng, mu: make(map[string]float64), sigma: make(map[string]float64)}
	for _, a := range agents {
		bs.mu[a.AgentID] = 0.5
		bs.sigma[a.AgentID] = 1.0
	}
	return bs
}

// Update — Bayesian (Kalman-filter style) belief update.
func (bs *BeliefState) Update(agentID string, success bool, reward float64) {
	obs := reward
	if !success && reward == 0 {
		obs = 0.0
	}
	mu, sigma := bs.mu[agentID], bs.sigma[agentID]
	noise := 0.1
	k := (sigma * sigma) / (sigma*sigma + noise*noise)
	bs.mu[agentID] = mu + k*(obs-mu)
	bs.sigma[agentID] = sigma * math.Sqrt(1-k)
	if bs.sigma[agentID] < 0.01 {
		bs.sigma[agentID] = 0.01
	}
}

// GetBelief — current belief snapshot for Thompson sampling.
func (bs *BeliefState) GetBelief() map[string]float64 {
	out := make(map[string]float64, len(bs.mu))
	for k, v := range bs.mu {
		out[k] = v
	}
	return out
}
