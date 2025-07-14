package rate

import "time"

type Limiter struct {
	limit float64
	rate  float64

	lastRefill time.Time
	tokens     float64
}

// rate refills limiter rate tokens per second
func NewLimiter(rate float64, limit float64) Limiter {
	return Limiter{
		limit:      limit,
		rate:       rate,
		lastRefill: time.Now().UTC(),
		tokens:     limit,
	}
}

func (rl *Limiter) Fill() {
	rl.update()
	rl.tokens = rl.limit
}

func (rl *Limiter) SetRate(rate float64) {
	rl.update()
	rl.rate = rate
}

func (rl *Limiter) SetLimit(limit float64) {
	rl.update()
	rl.limit = limit
}

func (rl *Limiter) Take(tokens float64) bool {
	rl.update()
	has := rl.Has(tokens)
	if has {
		rl.tokens -= tokens
		return true
	}
	return false
}

func (rl *Limiter) Has(tokens float64) bool {
	rl.update()
	return rl.tokens >= tokens
}

func (rl *Limiter) update() {
	lastRefill := rl.lastRefill
	rl.lastRefill = time.Now().UTC()

	rl.tokens += min(time.Since(lastRefill).Seconds()*rl.rate, rl.limit)
}
