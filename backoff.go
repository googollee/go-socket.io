package socketio

import (
	"math"
	"math/rand"
)

type BackOff struct {
	ms       float64
	max      float64
	factor   float64
	jitter   float64
	attempts float64
}

func NewBackOff(opts BackOff) *BackOff {
	return &BackOff{
		ms:       opts.ms,
		max:      opts.max,
		factor:   opts.factor,
		jitter:   opts.jitter,
		attempts: opts.attempts,
	}
}

func (b *BackOff) Duration() float64 {
	ms := b.ms * math.Pow(b.factor, b.attempts)
	b.attempts++

	if b.jitter > 0 {
		randVal := rand.Float64()
		deviation := math.Floor(randVal * b.jitter * ms)
		jitterDecision := int(math.Floor(randVal*10)) & 1
		if jitterDecision == 0 {
			ms -= deviation
		} else {
			ms += deviation
		}
	}

	return math.Min(ms, b.max)
}

func (b *BackOff) Reset() {
	b.attempts = 0
}

func (b *BackOff) SetMin(ms float64) {
	b.ms = ms
}

func (b *BackOff) SetMax(max float64) {
	b.max = max
}
func (b *BackOff) SetJitter(jitter float64) {
	b.jitter = jitter
}
